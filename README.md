# SMTP Server

A full-featured SMTP/IMAP mail server written in Go with built-in support for DKIM signing, SPF verification, spam filtering, and persistent storage via MySQL.

## Features

- **SMTP Server** — Receive and queue incoming mail with TLS support
- **IMAP Server** — Full IMAP4rev1 implementation for mailbox access
- **Outgoing Queue** — Automatic mail delivery with configurable retry logic and concurrent processing
- **DKIM Signing** — Cryptographic signing of outgoing messages
- **Authentication** — SASL authentication for both SMTP and IMAP
- **Spam Filtering** — Optional spam detection with separate spam mailbox
- **TLS Security** — Mandatory TLS (1.2+) for all connections
- **Persistent Storage** — MySQL backend for all messages and mailbox data
- **Logging** — Rotating file-based logging with slog

## Getting Started

### Prerequisites

- Go 1.25.4 or later
- MySQL 5.7 or later
- TLS certificates (public and private key in PEM format)
- Optional: RSA private key for DKIM signing (PKCS8 format)

### Building

```bash
go build -o smtp-server-go ./main.go
```

Or with Docker:

```bash
docker build -t smtp-server:local .
```

### Configuration

The server requires a JSON configuration file. Set the path via the `CONFIG_FILE` environment variable:

```bash
export CONFIG_FILE=/etc/smtp-server/config.json
./smtp-server-go
```

#### Configuration Schema

```json
{
  "domain": "mail.example.com",
  "listenAddress": "0.0.0.0:25",
  "address587": "0.0.0.0:587",
  "enabled587": true,
  "readTimeoutSecond": 10,
  "writeTimeoutSecond": 10,
  "maxMessageBytes": 10485760,
  "allowInsecureAuth": false,
  "certificates": {
    "public": "/etc/smtp-server/cert.pem",
    "private": "/etc/smtp-server/key.pem"
  },
  "imapAddress": "0.0.0.0:993",
  "db": {
    "machine": "localhost",
    "port": 3306,
    "login": "smtp_user",
    "password": "password",
    "database": "smtp_server"
  },
  "dkim": {
    "enabled": true,
    "selector": "default",
    "privateKey": "/etc/smtp-server/dkim.key"
  },
  "log": {
    "filename": "/var/log/smtp-server/server.log",
    "maxSize": 100,
    "maxAge": 30,
    "compress": true,
    "maxBackups": 10
  },
  "queue": {
    "timeBetweenLoads": 5,
    "loadSize": 100,
    "maxAttempts": 5,
    "timeBetweenAttempts": 300,
    "simultaneousProcessing": 10
  },
  "spam": {
    "enabled": true,
    "copy": true,
    "address": "spam@example.com",
    "threshold": 5.0
  }
}
```

**Key Configuration Notes:**

- **domain**: The mail server's domain (used in SMTP responses)
- **listenAddress**: Main SMTP listener (usually port 25)
- **address587**: Submission port for authenticated clients (optional)
- **certificates**: TLS certificates for secure connections
- **imapAddress**: IMAP server listener
- **dkim**: Set `enabled: false` to disable DKIM signing
- **queue**: Outgoing mail processing settings
- **spam**: Optional spam filtering; set `enabled: false` to disable

## Architecture

This project comprises several interconnected services:

- **SMTP Backend** — Receives mail, performs authentication, applies spam filtering
- **IMAP Backend** — Provides mailbox access and message retrieval
- **Queue Processor** — Delivers outgoing messages with retry logic and DKIM signing
- **Database Layer** — Persistent storage of messages, mailboxes, and user data

For detailed architectural information, see [CLAUDE.md](./CLAUDE.md).

## Development

### Building with Go

```bash
go build -o smtp-server-go ./main.go
```

### Docker Build

The project includes a Dockerfile for containerized deployment.

```bash
# Build
docker build -t smtp-server:local .

# Run (requires config volume)
docker run -v /path/to/config.json:/config.json \
  -e CONFIG_FILE=/config.json \
  -p 25:25 -p 587:587 -p 993:993 \
  smtp-server:local
```

## License

See [LICENSE](./LICENSE) file for details.
