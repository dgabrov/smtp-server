package start

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

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

	return proceedMainPort(config, db)
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

func proceedMainPort(config data.ConfigData, db *sql.DB) error {
	// 1. Setup Backend (The logic for auth and mail handling)
	// You must implement the 'Backend' and 'Session' interfaces
	server := srv.NewServer(db)
	be := bck.NewBackend(server)

	// 2. Initialize the Server
	s := smtp.NewServer(be)
	defer s.Close()

	// Hardcoded Machine and Port
	s.Domain = config.Domain
	s.ReadTimeout = time.Duration(config.ReadTimeoutSecond) * time.Second
	s.WriteTimeout = time.Duration(config.WriteTimeoutSecond) * time.Second
	s.MaxMessageBytes = config.MaxMessageBytes

	// 3. Mandatory TLS Configuration
	cert, err := tls.LoadX509KeyPair(
		config.Certificates.Public,
		config.Certificates.Private,
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
	s.AllowInsecureAuth = config.AllowInsecureAuth

	address := config.ListenAddress
	slog.Info(fmt.Sprintf("Starting SMTP Server on %s", address))

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	return s.Serve(l)

}
