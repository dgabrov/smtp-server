package imp

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/dgb9/smtp-server/internal/data"
	"github.com/dgb9/smtp-server/internal/srv"
	"github.com/emersion/go-imap/v2/imapserver"
)

func StartImap(config data.ConfigData, wg *sync.WaitGroup, tlsConfig *tls.Config, server srv.Servr) {
	err := start(config, wg, tlsConfig, server)

	if err != nil {
		slog.Error(err.Error())
	}
}

func start(config data.ConfigData, wg *sync.WaitGroup, tlsConfig *tls.Config, server srv.Servr) error {
	defer wg.Done()

	backend := newBackend(tlsConfig, server)
	backend.DebugWriter = os.Stdout

	s := imapserver.New(backend) // 'nil' should be replaced with your backend

	defer s.Close()

	// 3. Start the TLS Listener on 993
	ln, err := tls.Listen("tcp", config.ImapAddress, tlsConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to listen on %s: %v", config.ImapAddress, err))

		return err
	}

	slog.Info(fmt.Sprintf("IMAP server running on %s (Implicit TLS)", config.ImapAddress))

	// 4. Serve the connections
	if err = s.Serve(ln); err != nil {
		return err
	}

	return nil

}
