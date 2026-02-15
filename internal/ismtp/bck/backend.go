package bck

import (
	"context"

	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

type backend struct {
	server srv.Servr
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	remoteIP := c.Conn().RemoteAddr().String()
	helo := c.Hostname()

	return &session{
		server:   b.server,
		ctx:      context.WithValue(context.Background(), "uuid", uuid.NewString()),
		conn:     c,
		to:       make(map[string]bool),
		remoteIP: remoteIP,
		helo:     helo,
	}, nil
}

func NewBackend(servr srv.Servr) smtp.Backend {
	return &backend{server: servr}
}
