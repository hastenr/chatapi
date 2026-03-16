# ChatAPI

A lightweight, multitenant chat service built in Go with SQLite, WebSocket support, and durable message delivery.

[![Documentation](https://img.shields.io/badge/docs-hugo-blue)](https://hastenr.github.io/chatapi/)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/hastenr/chatapi)](https://github.com/hastenr/chatapi/releases)
[![Docker Image Version (latest by date)](https://img.shields.io/docker/v/hastenr/chatapi)](https://hub.docker.com/r/hastenr/chatapi)

## Releases

Download pre-built binaries and Docker images from [GitHub Releases](https://github.com/hastenr/chatapi/releases).

### Binary Installation

```bash
# Download latest release for your platform
curl -L https://github.com/hastenr/chatapi/releases/latest/download/chatapi-linux-amd64.tar.gz | tar xz
sudo mv chatapi /usr/local/bin/
```

### Docker Installation

```bash
# Pull from Docker Hub
docker pull hastenr/chatapi:latest

# Run with environment variables
docker run -p 8080:8080 -e MASTER_API_KEY=your-key hastenr/chatapi
```

## Features

- **Multitenant**: API key-based tenancy with per-tenant rate limiting
- **Real-time messaging**: WebSocket connections for instant chat
- **Durable delivery**: Store-then-send with at-least-once guarantees
- **Message sequencing**: Per-room message ordering with acknowledgments
- **Room metadata**: Attach arbitrary app-level context (listing IDs, order IDs, etc.) to rooms at creation time
- **Offline webhooks**: POST webhook notifications to your backend when a message arrives for an offline user
- **SQLite backend**: WAL mode for concurrent reads/writes
- **REST & WebSocket APIs**: Complete HTTP and real-time interfaces

## Quick Start

### Prerequisites

- Go 1.22+
- SQLite3 (optional, for CGO builds)

### Installation

```bash
# Clone the repository
git clone https://github.com/hastenr/chatapi.git
cd chatapi

# Install dependencies
go mod download

# Build the binary
go build ./cmd/chatapi

# Or use Makefile
make build

# Build for all platforms
make build-all
```

### Docker

```bash
# Build the Docker image
docker build -t chatapi .

# Or use Makefile
make docker-build

# Run the container
docker run -p 8080:8080 -e MASTER_API_KEY=your-key chatapi

# Or use Makefile
make docker-run
```

### Run

```bash
# Set required environment variables
export LISTEN_ADDR=":8080"
export DATABASE_DSN="file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000"

# Start the server
./chatapi
```

### Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"chatapi","uptime":"1m30s","db_writable":true}
```

## Documentation

📚 **[Complete Documentation](https://hastenr.github.io/chatapi/)**

- **Getting Started**: Installation and setup guides
- **API Reference**: REST and WebSocket API documentation
- **Guides**: Advanced usage and integration examples
- **Architecture**: System design and database schema
- **API Playground**: Interactive API testing

### Local Documentation

```bash
# Install Hugo
sudo snap install hugo

# Start docs server
cd docs && hugo server

# Visit http://localhost:1313
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MASTER_API_KEY` | *(required)* | Master API key for admin operations — server will not start without this |
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATABASE_DSN` | `file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000` | SQLite database DSN |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `DEFAULT_RATE_LIMIT` | `100` | Requests per second per tenant |
| `WS_ALLOWED_ORIGINS` | *(none)* | Comma-separated allowed origins for WebSocket connections and REST CORS headers. Use `*` for dev only. Unset = reject all browser-origin connections |
| `MAX_CONNECTIONS_PER_USER` | `5` | Max concurrent WebSocket connections per user |
| `DATA_DIR` | `/var/chatapi` | Directory for data files |
| `LOG_DIR` | `/var/log/chatapi` | Directory for log files |
| `WORKER_INTERVAL` | `30s` | Background worker interval |
| `RETRY_MAX_ATTEMPTS` | `5` | Max delivery retry attempts |
| `RETRY_INTERVAL` | `30s` | Retry interval |
| `SHUTDOWN_DRAIN_TIMEOUT` | `10s` | Graceful shutdown timeout |

See [Configuration Guide](https://hastenr.github.io/chatapi/getting-started/) for all options.

## API Example

### Create a Tenant (Admin)

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Master-Key: your-master-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "MyCompany"}'
```

The response includes the `api_key` field — this is the only time the plaintext key is returned. Store it immediately in a secrets manager; it cannot be retrieved again.

### Create a Room

```bash
curl -X POST http://localhost:8080/rooms \
  -H "X-API-Key: your-api-key" \
  -H "X-User-Id: user123" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "dm",
    "members": ["alice", "bob"],
    "metadata": "{\"listing_id\":\"lst_99\",\"order_id\":\"ord_42\"}"
  }'
```

Use `metadata` to attach app-level context to the room (listing IDs, order IDs, etc.). The value is an arbitrary JSON string stored as-is and returned on every room object.

### Send a Message

```bash
curl -X POST http://localhost:8080/rooms/room_123/messages \
  -H "X-API-Key: your-api-key" \
  -H "X-User-Id: user123" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello, World!"}'
```

### Connect via WebSocket (Browser)

```javascript
// Step 1: get a short-lived token
const { token } = await fetch('/ws/token', {
  method: 'POST',
  headers: { 'X-API-Key': apiKey, 'X-User-Id': userId }
}).then(r => r.json());

// Step 2: connect with the token
const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${token}`);
```

### Connect via WebSocket (Node.js / server)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws', [], {
  headers: { 'X-API-Key': apiKey, 'X-User-Id': userId }
});
```

### Tenant Config (webhook_url)

To receive offline delivery webhooks, set `webhook_url` in the tenant's config:

```bash
# Set via your admin tooling or directly in the DB config column:
# { "webhook_url": "https://your-app.example.com/chatapi-webhook" }
```

When a message arrives for a user with no active WebSocket connection, ChatAPI will POST the message payload to that URL so your backend can trigger push notifications or other offline delivery.

## Architecture

```
cmd/chatapi/           # Application entry point
internal/
  config/              # Configuration management
  db/                  # Database connection and migrations
  models/              # Data structures
  services/            # Business logic (tenant, chat, realtime, etc.)
  handlers/            # HTTP and WebSocket handlers
  transport/           # HTTP server and graceful shutdown
  worker/              # Background workers
```

## Development

```bash
# Run tests
go test ./...

# Build with race detection
go build -race ./cmd/chatapi

# Debug logging
export LOG_LEVEL=debug && ./chatapi
```

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o chatapi ./cmd/chatapi
CMD ["./chatapi"]
```

### Systemd

```ini
[Unit]
Description=ChatAPI Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/chatapi
Restart=always

[Install]
WantedBy=multi-user.target
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
