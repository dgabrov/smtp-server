package bck

import (
	"strings"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/logger"
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

type backend struct {
	server srv.Servr
	spam   data.Spam
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	remoteIP := c.Conn().RemoteAddr().String()

	// if the index of : is
	index := strings.Index(remoteIP, ":")
	if index >= 0 {
		remoteIP = remoteIP[:index]
	}

	helo := c.Hostname()

	newID := uuid.NewString()
	ctx := logger.GetLogContext(newID)

	return &session{
		server:   b.server,
		ctx:      ctx,
		conn:     c,
		to:       make(map[string]bool),
		remoteIP: remoteIP,
		helo:     helo,
		spam:     b.spam,
	}, nil
}

func NewBackend(servr srv.Servr, spam data.Spam) smtp.Backend {
	return &backend{
		server: servr,
		spam:   spam,
	}
}
