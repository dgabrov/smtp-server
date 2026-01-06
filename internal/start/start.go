package start

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dgb9/smtp-server/internal/queue"
	"github.com/dgb9/smtp-server/internal/srv"
	_ "github.com/go-sql-driver/mysql"

	"github.com/dgb9/smtp-server/internal/bck"
	"github.com/dgb9/smtp-server/internal/data"
	"github.com/emersion/go-smtp"
)

func Start() error {

	config, err := data.LoadConfig()
	if err != nil {
		return err
	}

	configureLogger(config)

	data.LogConfig(config)

	dbConfig := config.Db
	db, err := getDatabaseConnectionPool(dbConfig.Machine, dbConfig.Port, dbConfig.Login, dbConfig.Password, dbConfig.Database)
	if err != nil {
		return err
	}

	server := srv.NewServer(db)
	be := bck.NewBackend(server)

	tlsConfig, err := loadCertificates(config)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2) // 25 is open no matter what and the loader is also processing
	if config.Enabled587 {
		wg.Add(1) // if enabled 587, add another one as well
	}

	go func() {
		err = proceedMainPort(config, &wg, be, tlsConfig, config.ListenAddress)

		if err != nil {
			slog.Error(fmt.Sprintf("error running main listener: %s", err.Error()))
		}
	}()

	if config.Enabled587 {
		go func() {
			err = proceedMainPort(config, &wg, be, tlsConfig, config.Address587)

			if err != nil {
				slog.Error(fmt.Sprintf("error running main listener: %s", err.Error()))
			}
		}()
	}

	go queue.StartQueue(&wg, config.Queue, server)

	wg.Wait()

	return nil
}

func loadCertificates(config data.ConfigData) (*tls.Config, error) {
	// 3. Mandatory TLS Configuration
	cert, err := tls.LoadX509KeyPair(
		config.Certificates.Public,
		config.Certificates.Private,
	)

	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Ensures we use modern, secure protocols
		MinVersion: tls.VersionTLS12,
	}, nil
}

func getDatabaseConnectionPool(machine string, port int, login string, password string, database string) (*sql.DB, error) {
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", login, password, machine, port, database)
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func proceedMainPort(config data.ConfigData, wg *sync.WaitGroup, be smtp.Backend, tlsConfig *tls.Config, listenAddress string) error {
	defer wg.Done()

	// 2. Initialize the Server
	s := smtp.NewServer(be)
	defer s.Close()

	// Hardcoded Machine and Port
	s.Domain = config.Domain
	s.ReadTimeout = time.Duration(config.ReadTimeoutSecond) * time.Second
	s.WriteTimeout = time.Duration(config.WriteTimeoutSecond) * time.Second
	s.MaxMessageBytes = config.MaxMessageBytes

	s.TLSConfig = tlsConfig

	// 4. Force TLS for Authentication
	// This prevents the server from accepting AUTH commands over plain text
	s.AllowInsecureAuth = config.AllowInsecureAuth

	slog.Info(fmt.Sprintf("Starting SMTP Server on %s", listenAddress))

	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return err
	}
	defer l.Close()

	return s.Serve(l)

}
