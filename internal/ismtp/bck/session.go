package bck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/Teamwork/spamc"
	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/mileusna/spf"
)

type session struct {
	login         string
	from          string
	to            map[string]bool // value is local or not, key is the email address
	authenticated bool
	ctx           context.Context
	server        srv.Servr
	conn          *smtp.Conn
	remoteIP      string
	helo          string
	spam          data.Spam
}

func (s *session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

func (s *session) Auth(mech string) (sasl.Server, error) {
	if mech != sasl.Plain {
		return nil, errors.New("only PLAIN is supported")
	}

	return sasl.NewPlainServer(
		func(identity, username, password string) error {
			s.login = username
			err := s.server.Authenticate(s.ctx, username, password)
			if err != nil {
				slog.InfoContext(s.ctx, fmt.Sprintf("INVALID authentication: %s", username))
				return err
			}

			s.authenticated = true
			slog.InfoContext(s.ctx, fmt.Sprintf("successfully authenticated: %s", username))

			return nil
		},
	), nil
}

func (s *session) Reset() {
	slog.InfoContext(s.ctx, "reset")

	s.login = ""
	s.from = ""
	clear(s.to)
}

func (s *session) Logout() error {
	slog.InfoContext(s.ctx, "logout")

	return nil
}

func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	// log the tls status
	_, ok := s.conn.TLSConnectionState()
	message := "current session is TLS"

	if !ok {
		message = "current session is NOT TLS"
	}

	slog.InfoContext(s.ctx, message)

	s.from = from
	slog.InfoContext(s.ctx, fmt.Sprintf("mail: %s", s.from))

	// now establish the spf if it is good or not
	ipAddress := net.ParseIP(s.remoteIP)
	domain, err := srv.GetDomain(from)
	if err != nil {
		slog.ErrorContext(s.ctx, fmt.Sprintf("error getting domain: %s", from))
	} else {
		res := spf.CheckHost(ipAddress, domain, "", s.helo)

		// now logging the result for spf processing
		strRes := res.String()

		slog.InfoContext(s.ctx, fmt.Sprintf("checking spf host: %s with ip address: %s, result: %s, helo: %s", domain, s.remoteIP, strRes, s.helo))
	}

	return nil
}

func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	slog.InfoContext(s.ctx, fmt.Sprintf("to: %s", to))

	// get usr and domain
	usr, domain, err := s.server.GetLoginAndDomain(s.ctx, to)
	slog.InfoContext(s.ctx, fmt.Sprintf("usr: %s, domain: %s", usr, domain))
	if err != nil {
		return err
	}

	local, err := s.server.IsLocalDomain(s.ctx, domain)
	slog.InfoContext(s.ctx, fmt.Sprintf("local: %t", local))

	if local {
		err = s.server.CheckLocalUserAndDomain(s.ctx, usr, domain)
		if err != nil {
			return err
		}

		slog.InfoContext(s.ctx, fmt.Sprintf("checked local user: %s and domain: %s", usr, domain))

		s.to[to] = true
	} else {
		if s.authenticated {
			// attach to queue
			slog.InfoContext(s.ctx, fmt.Sprintf("will relay, as it is authenticated: %s", s.login))

			s.to[to] = false
		} else {
			slog.InfoContext(s.ctx, fmt.Sprintf("will not relay: must be authenticated: %s", s.login))

			return fmt.Errorf("%s not local domain, cannot relay with unauthenticated user %s", domain, s.login)
		}
	}

	return nil
}

func (s *session) Data(r io.Reader) error {
	slog.InfoContext(s.ctx, "data")

	bytes, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	var relay []string

	for email, local := range s.to {
		if local {
			// check if dkim is ok, if not, no issue, but at least write it down

			s.processDkimInbound(bytes)

			move, score := s.processSpam(bytes)

			err = s.server.DeliverLocally(s.ctx, email, bytes, move, score)

			if err != nil {
				slog.ErrorContext(s.ctx, fmt.Sprintf("error delivering locally delivery locally email: %s, error: %s", email, err.Error()))
			}
		} else {
			relay = append(relay, email)
		}
	}

	// if the relay is larger than zero, then we address this
	return s.server.DeliverQueue(s.ctx, s.from, relay, bytes)
}

func (s *session) processDkimInbound(bytes []byte) {
	if s.authenticated {
		slog.InfoContext(s.ctx, "dkim inbound we do not check dkim because this is authenticated session")
	} else {
		strMessage := string(bytes)
		messageReader := strings.NewReader(strMessage)
		verifications, err := dkim.Verify(messageReader)
		if err != nil {
			slog.ErrorContext(s.ctx, fmt.Sprintf("dkim inbound verification error: %s", err.Error()))
		} else {
			if len(verifications) == 0 {
				slog.InfoContext(s.ctx, "dkim inbound there is no dkim signature attached")
			}

			for _, verification := range verifications {

				if verification.Err == nil {
					slog.InfoContext(s.ctx, fmt.Sprintf("dkim inbound VALID: Signature for domain %s passed.", verification.Domain))
				} else {
					slog.ErrorContext(s.ctx, fmt.Sprintf("dkim inbound INVALID: Domain %s failed with error: %v", verification.Domain, verification.Err.Error()))
				}
			}
		}

	}
}

func (s *session) processSpam(msg []byte) (bool, float64) {
	move := false
	score := 0.0

	spamConfig := s.spam

	if !spamConfig.Enabled || s.authenticated {
		// either not enabled or authenticated
		return move, score
	}

	client := spamc.New(spamConfig.Address, &net.Dialer{Timeout: 5 * time.Second})
	result, err := client.Check(s.ctx, bytes.NewReader(msg), nil)

	if err != nil {
		slog.ErrorContext(s.ctx, fmt.Sprintf("error checking spam result: %s", err.Error()))
	} else {
		score = result.ResponseScore.Score

		slog.Info(fmt.Sprintf("message spam score: %5.2f", score))

		if score > spamConfig.Threshold && spamConfig.Copy {
			move = true
		}
	}

	return move, score
}
