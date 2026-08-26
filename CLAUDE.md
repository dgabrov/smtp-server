# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

### Building
```bash
# Build the binary directly
go build -o smtp-server-go ./main.go

# Build using Docker (matches CI)
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 docker run -v $(pwd):/alfa -w /alfa golang:1.25.5-alpine3.23 go build -o smtp-server-go ./main.go

# Build Docker image
docker build . -t smtp-server:local
```

### Running
The application requires a configuration file passed via the `CONFIG_FILE` environment variable (JSON format):
```bash
CONFIG_FILE=/path/to/config.json ./smtp-server-go
```

Configuration includes: SMTP domain, listen address, TLS certificates, database connection, DKIM settings, logging, mail queue settings, IMAP address, and spam filtering.

### Go Version
- Requires Go 1.25.4+
- Uses Go modules (go.mod)

## Architecture Overview

This is a combined SMTP/IMAP server with mail queueing and advanced features (DKIM, SPF, spam filtering).

### Core Components

**Initialization & Config** (`internal/data`, `internal/start`)
- `config.go`: JSON configuration schema with database, TLS, DKIM, queue, logging settings
- `start.go`: Main entry point that initializes all subsystems (database, TLS, DKIM key loading, listeners, queue, IMAP)

**SMTP Server** (`internal/ismtp/`)
- `bck/backend.go`: SMTP backend implementation - handles authentication, message receipt, spam filtering
- `bck/session.go`: Per-connection SMTP session state
- `queue/`: Outgoing mail processor
  - `queue.go`: Main queue loop - loads messages from DB, processes them
  - `proc.go`: Individual message processing with retries
  - `dkim.go`: DKIM signing logic

**IMAP Server** (`internal/imp/`)
- `imapbackend.go`: IMAP backend implementation
- `session.go`: Per-connection IMAP session
- `fetch.go`, `copy.go`, `search.go`: Core IMAP operations

**Message Storage** (`internal/srv/`)
- `server.go`: Core server interface managing mailboxes and users
- `mailbox.go`: Mailbox operations (create, delete, list)
- `message.go`, `message_flags.go`: Message storage and flag handling
- Uses MySQL database for persistence

**Logging** (`internal/logger/`)
- Uses `log/slog` (Go 1.21+)
- Configured with lumberjack for rotating file logs

### Data Flow

1. **Incoming Mail**: SMTP listener → backend session → database storage → spam filtering (optional)
2. **Outgoing Mail**: Messages in DB queue → processor loads in batches → DKIM signing → delivery attempts with retries
3. **IMAP Access**: Client session → IMAP backend → mailbox queries from database

### Database
- MySQL (driver: `github.com/go-sql-driver/mysql`)
- Connection pool: 25 open, 25 idle, 5-minute max lifetime
- Must be accessible before server starts

### TLS & Security
- Mandatory TLS with min version TLS 1.2
- Separate listeners on port 25 (SMTP) and 587 (submission) if enabled
- DKIM signing for outgoing mail (RSA private key loaded at startup)
- SPF/DKIM verification via dependencies

## Key Dependencies

- `github.com/emersion/go-smtp`: SMTP protocol
- `github.com/emersion/go-imap/v2`: IMAP protocol
- `github.com/emersion/go-msgauth`: Message authentication (DKIM)
- `github.com/emersion/go-sasl`: SASL authentication
- `gopkg.in/natefinch/lumberjack.v2`: Log rotation
- `github.com/go-sql-driver/mysql`: Database driver

## Important Notes

- The server runs multiple concurrent listeners (SMTP, IMAP, queue processor) in separate goroutines
- Configuration must be provided as a JSON file; no environment variable overrides for individual settings
- DKIM key must be in PKCS8 PEM format
- Graceful shutdown via SIGTERM/SIGINT
- Spam filtering is optional and configurable (includes copy-to-spam-folder feature)

## Deployment

Built and deployed as Docker containers to AWS ECR:
- Uses Alpine Linux base
- Expose ports 8225 (SMTP), 8587 (submission), 8993 (IMAP)
- Pre-built and pushed to ECR on manual GitHub Actions workflow trigger
