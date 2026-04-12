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
git clone https://github.com/getchatapi/chatapi.git
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
| `ALLOWED_ORIGINS` | *(none)* | Comma-separated allowed origins for CORS and WebSocket upgrade. Required for browser clients. Use `*` for local dev. |
| `WEBHOOK_URL` | *(none)* | Your backend URL for webhook events. **Required if you use bots** (provides the system prompt). Also called for offline push notifications. |
| `WEBHOOK_SECRET` | *(none)* | HMAC-SHA256 secret for verifying webhook calls from ChatAPI. Recommended in production. |
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATABASE_DSN` | `file:./chatapi.db` (binary) / `file:/data/chatapi.db` (Docker) | SQLite connection string. The Docker image defaults to `/data` — mount a volume there to persist data across restarts. |
| `RATE_LIMIT_MESSAGES` | `10` | Sustained message sends per second per user. Set to `0` to disable. |
| `RATE_LIMIT_MESSAGES_BURST` | `20` | Burst allowance on top of the sustained rate. |
| `GEMINI_API_KEY` / `OPENAI_API_KEY` / … | *(none)* | LLM API keys referenced by managed bots via `llm_api_key_env`. Name the variable whatever you like — the bot config stores only the variable name, not the key. |

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
export DATABASE_DSN="file:./data/chatapi.db"
```

**Build errors:**
```bash
go clean
go mod tidy
go build ./cmd/chatapi
```
