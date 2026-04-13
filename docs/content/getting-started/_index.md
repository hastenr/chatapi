---
title: "Getting Started"
weight: 10
---

# Getting Started

## Install

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -e ALLOWED_ORIGINS="*" \
  -v chatapi-data:/data \
  hastenr/chatapi:latest
```

For production deployment with Docker Compose, TLS, and reverse proxy, see the [Deploy guide](/deploy/).

### Health check

```bash
curl http://localhost:8080/health
# {"status":"ok","db_writable":true}
```

---

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | **required** | HS256 secret for validating Bearer tokens. Generate: `openssl rand -base64 32` |
| `ALLOWED_ORIGINS` | *(none)* | Comma-separated CORS origins. Required for browser clients. Use `*` for local dev. |
| `WEBHOOK_URL` | *(none)* | Required if you use bots — ChatAPI calls this before every LLM request and for offline push notifications. |
| `WEBHOOK_SECRET` | *(none)* | HMAC-SHA256 secret — ChatAPI signs webhook requests with this. Verify the `X-ChatAPI-Signature` header on your end. |
| `LISTEN_ADDR` | `:8080` | Server listen address |
| `DATABASE_DSN` | `file:./chatapi.db` (binary) / `file:/data/chatapi.db` (Docker) | SQLite connection string. Mount a volume at `/data` in Docker to persist data. |
| `RATE_LIMIT_MESSAGES` | `10` | Sustained message sends per second per user. `0` to disable. |
| `RATE_LIMIT_MESSAGES_BURST` | `20` | Burst allowance on top of the sustained rate. |
| `WS_MAX_CONNECTIONS_PER_USER` | `5` | Max concurrent WebSocket connections per user. |
| `WORKER_INTERVAL` | `30s` | How often the delivery worker retries undelivered messages. |
| `SHUTDOWN_DRAIN_TIMEOUT` | `10s` | Graceful shutdown timeout. |

LLM API keys (e.g. `GEMINI_API_KEY`, `OPENAI_API_KEY`) are not ChatAPI config — they are arbitrary env var names you reference when registering a bot via `llm_api_key_env`.

---

## Authentication

Your backend mints HS256 JWTs signed with `JWT_SECRET`. The `sub` claim is the user ID. ChatAPI never issues tokens — it only validates them.

```
Authorization: Bearer <token>          # REST
ws://localhost:8080/ws?token=<token>   # WebSocket (browser)
```

See [Authentication](/api/rest/#authentication) in the API reference for code examples.

---

## First API call

```bash
TOKEN="<your-signed-jwt>"

# Create a DM room between two users
curl -X POST http://localhost:8080/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "dm", "members": ["alice", "bob"]}'
```

```json
{
  "room_id": "room_abc123",
  "type": "dm",
  "last_seq": 0,
  "created_at": "2026-04-12T10:00:00Z"
}
```

---

## Next steps

- [REST API](/api/rest/) — Full endpoint reference
- [WebSocket API](/api/websocket/) — All events and client messages
- [AI Bots](/guides/bots/) — Register a bot and add it to a room
- [Deploy](/deploy/) — Production setup with TLS and reverse proxy
