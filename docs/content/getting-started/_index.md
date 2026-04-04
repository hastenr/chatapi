---
title: "Getting Started"
weight: 10
---

# Getting Started with ChatAPI

## Prerequisites

- Go 1.22+
- CGO-enabled build toolchain (for the SQLite driver)

## Installation

### Build from source

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
go mod download
go build -o bin/chatapi ./cmd/chatapi
```

### Docker

```bash
docker pull hastenr/chatapi:latest
```

## Configuration

ChatAPI is configured entirely via environment variables. Copy `.env.example` and fill in the required values:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(required)* | Secret used to validate JWT Bearer tokens. Generate with `openssl rand -base64 32`. |
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATABASE_DSN` | `file:chatapi.db?_journal_mode=WAL&_busy_timeout=5000` | SQLite connection string |
| `ALLOWED_ORIGINS` | *(none)* | Comma-separated allowed origins for CORS and WebSocket upgrade. Use `*` for local dev. |
| `WS_MAX_CONNECTIONS_PER_USER` | `5` | Maximum concurrent WebSocket connections per user |
| `WORKER_INTERVAL` | `30s` | Background delivery worker interval |
| `SHUTDOWN_DRAIN_TIMEOUT` | `10s` | Graceful shutdown drain timeout |

## Start the server

```bash
export JWT_SECRET=$(openssl rand -base64 32)
export ALLOWED_ORIGINS="*"
./bin/chatapi
```

Expected output:

```
time=2026-04-02T12:00:00Z level=INFO msg="starting server" addr=:8080
time=2026-04-02T12:00:00Z level=INFO msg="delivery worker started" interval=30s
```

### Health check

```bash
curl http://localhost:8080/health
```

```json
{"status": "ok", "db_writable": true}
```

## Authentication

ChatAPI uses JWT Bearer tokens. Your backend signs JWTs with `JWT_SECRET`; ChatAPI validates the signature and reads the `sub` claim as the user ID.

**Mint a token for testing** (Go):

```go
package main

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

func mintToken(secret, userID string) (string, error) {
    claims := jwt.MapClaims{
        "sub": userID,
        "exp": time.Now().Add(24 * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

func main() {
    tok, _ := mintToken("your-jwt-secret", "alice")
    fmt.Println(tok)
}
```

Or use any JWT library in your language of choice — the token just needs a `sub` claim and must be signed with HS256 using `JWT_SECRET`.

## First REST call

```bash
TOKEN="<your-signed-jwt>"

# Create a DM room between alice and bob
curl -X POST http://localhost:8080/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "dm", "members": ["alice", "bob"]}'
```

```json
{
  "room_id": "room_abc123",
  "type": "dm",
  "name": null,
  "metadata": null,
  "last_seq": 0,
  "created_at": "2026-04-02T12:00:00Z"
}
```

## First WebSocket connection

**Browser:**

```javascript
const token = "<your-signed-jwt>";
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg);
};
```

**Server / Node.js client:**

```javascript
const WebSocket = require("ws");

const ws = new WebSocket("ws://localhost:8080/ws", {
  headers: { Authorization: "Bearer <your-signed-jwt>" },
});

ws.on("message", (data) => {
  console.log(JSON.parse(data));
});
```

After connecting, send a message to a room:

```javascript
ws.send(JSON.stringify({
  type: "send_message",
  data: { room_id: "room_abc123", content: "Hello!" },
}));
```

## Next Steps

- [REST API Reference](/api/rest/) — Full endpoint documentation
- [WebSocket API Reference](/api/websocket/) — All events and client messages
- [AI Bots](/guides/bots/) — Add an LLM-backed bot to a room
- [Architecture](/architecture/) — How ChatAPI is structured internally

## Troubleshooting

**Port already in use:**
```bash
export LISTEN_ADDR=":3000"
```

**Database permission errors:**
```bash
mkdir -p ./data
chmod 755 ./data
export DATABASE_DSN="file:./data/chatapi.db?_journal_mode=WAL&_busy_timeout=5000"
```

**Build errors:**
```bash
go clean
go mod tidy
go build ./cmd/chatapi
```
