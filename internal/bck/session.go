package bck

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

type session struct {
	from string
	to   string
	ctx  context.Context
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
			return nil
		},
	), nil
}

func (s *session) Reset() {
	slog.Info("reset")
}

func (s *session) Logout() error {
	slog.Info("logout")
	return nil
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	s.ctx = context.WithValue(context.Background(), "guid", uuid.NewString())
	slog.InfoContext(s.ctx, "mail from", slog.String("mail", s.from))

	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = to
	slog.InfoContext(s.ctx, "to", s.to)

	return nil
}

func (s *session) Data(r io.Reader) error {
	slog.InfoContext(s.ctx, "data")

	return nil
}
