package bck

import (
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-smtp"
)

type backend struct {
	server srv.Servr
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{
		server: b.server,
	}, nil
}

func NewBackend(servr srv.Servr) smtp.Backend {
	return &backend{server: servr}
}
