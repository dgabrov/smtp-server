package bck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type session struct {
	login         string
	from          string
	to            string
	local         bool
	authenticated bool
	ctx           context.Context
	server        srv.Servr
	conn          *smtp.Conn
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
	s.to = ""
	s.local = false
}

func (s *session) Logout() error {
	slog.InfoContext(s.ctx, "logout")

	return nil
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	// log the tls status
	_, ok := s.conn.TLSConnectionState()
	message := "current session is TLS"

	if !ok {
		message = "current session is NOT TLS"
	}

	slog.InfoContext(s.ctx, message)

	s.from = from
	slog.InfoContext(s.ctx, fmt.Sprintf("mail: %s", s.from))

	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = to
	slog.InfoContext(s.ctx, fmt.Sprintf("to: %s", s.to))

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

		s.local = true
	} else {
		if s.authenticated {
			// attach to queue
			slog.InfoContext(s.ctx, fmt.Sprintf("will relay, as it is authenticated: %s", s.login))

			s.local = false
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

	if s.local {
		err = s.server.DeliverLocally(s.ctx, s.to, bytes)
	} else {
		err = s.server.DeliverQueue(s.ctx, s.from, s.to, bytes)
	}

	return err
}
