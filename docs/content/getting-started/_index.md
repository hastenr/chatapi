---
title: "Getting Started"
weight: 10
---

# Getting Started with ChatAPI

Welcome to ChatAPI! This guide will help you get up and running with your own chat service instance.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.22 or later** - [Download from golang.org](https://golang.org/dl/) (required for enhanced `net/http` routing)
- **Git** - For cloning the repository
- **SQLite3** (optional) - For CGO builds with native SQLite driver

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Application

```bash
# Build with modernc SQLite driver (recommended)
go build -o bin/chatapi ./cmd/chatapi

# Or build with CGO SQLite driver (if you have SQLite3 installed)
CGO_ENABLED=1 go build -o bin/chatapi ./cmd/chatapi
```

## Configuration

ChatAPI uses environment variables for configuration. Create a `.env` file or set them directly:

```bash
# Server configuration
export LISTEN_ADDR=":8080"
export DATABASE_DSN="file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000"

# Admin configuration (required for tenant creation)
export MASTER_API_KEY="your-secure-master-key-here"

# Optional: Database directory
export DATA_DIR="./data"

# Optional: Logging level
export LOG_LEVEL="info"

# CORS and WebSocket origin allowlist (use * for local dev only)
export WS_ALLOWED_ORIGINS="*"

# Maximum concurrent WebSocket connections per user
export MAX_CONNECTIONS_PER_USER=5
```

### Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATABASE_DSN` | `file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000` | SQLite database connection string |
| `MASTER_API_KEY` | *(required)* | Master API key for admin operations — server will not start without this |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `DEFAULT_RATE_LIMIT` | `100` | Requests per second per tenant |
| `WS_ALLOWED_ORIGINS` | *(none)* | Comma-separated allowed origins for WebSocket connections and REST CORS headers (e.g. `https://app.example.com`). Use `*` for dev only. Unset = reject all browser-origin connections |
| `MAX_CONNECTIONS_PER_USER` | `5` | Maximum concurrent WebSocket connections per user |
| `DATA_DIR` | `/var/chatapi` | Directory for data files |
| `LOG_DIR` | `/var/log/chatapi` | Directory for log files |
| `WORKER_INTERVAL` | `30s` | Background worker interval |
| `RETRY_MAX_ATTEMPTS` | `5` | Max delivery retry attempts |
| `RETRY_INTERVAL` | `30s` | Retry interval |
| `SHUTDOWN_DRAIN_TIMEOUT` | `10s` | Graceful shutdown timeout |

## Running ChatAPI

### Start the Server

```bash
./bin/chatapi
```

You should see output similar to:
```
2025/12/13 12:00:00 Starting ChatAPI server addr=:8080
2025/12/13 12:00:00 Starting delivery worker interval=30s
2025/12/13 12:00:00 Starting WAL checkpoint worker interval=5m0s
2025/12/13 12:00:00 Starting HTTP server addr=:8080
```

### Health Check

Verify the server is running:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "uptime": "1m30s",
  "db_writable": true
}
```

### Create Your First Tenant

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Master-Key: your-secure-master-key-here" \
  -H "Content-Type: application/json" \
  -d '{"name": "MyApp"}'
```

The response includes an `api_key` field. **This is the only time the plaintext API key is returned** — it is stored as a SHA-256 hash in the database and cannot be retrieved again. Copy it immediately and store it in a secrets manager or environment variable.

```json
{
  "tenant_id": "tenant_abc123",
  "name": "MyApp",
  "api_key": "sk_abc123def456...",
  "created_at": "2025-12-13T12:00:00Z"
}
```

## Next Steps

Now that you have ChatAPI running, you can:

1. **[Create a Tenant](/guides/tenants/)** - Set up your first tenant with API keys
2. **[Create Rooms](/guides/rooms/)** - Start creating chat rooms
3. **[Send Messages](/guides/messaging/)** - Begin messaging
4. **[Integrate WebSockets](/guides/websockets/)** - Add real-time functionality

## Development Mode

For development with live reloading:

```bash
# Run with debug logging
export LOG_LEVEL="debug"
./bin/chatapi

# Or use air for hot reloading (if installed)
air
```

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Change the port
export LISTEN_ADDR=":3000"
```

**Database permission errors:**
```bash
# Ensure write permissions to data directory
mkdir -p ./data
chmod 755 ./data
```

**Build errors:**
```bash
# Clean and rebuild
go clean
go mod tidy
go build ./cmd/chatapi
```

### Logs

Check the structured JSON logs for debugging:
```bash
./bin/chatapi 2>&1 | jq .
```

For more help, check the [troubleshooting guide](/guides/troubleshooting/) or [open an issue](https://github.com/hastenr/chatapi/issues).
