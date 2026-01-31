package imp

import (
	"crypto/tls"

	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-imap/v2/imapserver"
)

func newBackend(tlsConfig *tls.Config, server srv.Servr) *imapserver.Options {
	return &imapserver.Options{
		TLSConfig:    tlsConfig,
		InsecureAuth: false,
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return newImapSession(server), &imapserver.GreetingData{
				PreAuth: false, // true means that the auth is done via certificate which is never the case in real life
			}, nil
		},
	}
}
