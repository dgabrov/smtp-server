package bck

import "github.com/emersion/go-smtp"

type backend struct {
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &session{}, nil
}

func NewBackend() smtp.Backend {
	return &backend{}
}
