package start

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/dgb9/smtp-server/internal/bck"
	"github.com/emersion/go-smtp"
)

func Start() error {
	// 1. Setup Backend (The logic for auth and mail handling)
	// You must implement the 'Backend' and 'Session' interfaces
	be := bck.NewBackend()

	// 2. Initialize the Server
	s := smtp.NewServer(be)
	defer s.Close()

	// Hardcoded Machine and Port
	s.Addr = "127.0.0.1:8225"
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024 // 1MB limit

	// 3. Mandatory TLS Configuration
	cert, err := tls.LoadX509KeyPair(
		"/home/daniel/IdeaProjects/smtp-server/docs/cert.pem",
		"/home/daniel/IdeaProjects/smtp-server/docs/key.pem",
	)

	if err != nil {
		return err
	}

	s.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Ensures we use modern, secure protocols
		MinVersion: tls.VersionTLS12,
	}

	// 4. Force TLS for Authentication
	// This prevents the server from accepting AUTH commands over plain text
	s.AllowInsecureAuth = false

	log.Printf("Starting SMTP Server on %s", s.Addr)
	log.Println("STARTTLS is mandatory for authentication")

	return s.ListenAndServe()
}
