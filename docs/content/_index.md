+++
title = "ChatAPI Documentation"
type = "book"
weight = 1
+++

# ChatAPI

The messaging layer for apps where AI is a participant.

Drop-in WebSocket rooms with LLM streaming baked into the messaging layer — no infrastructure to manage beyond a single binary. Self-hosted, open source.

## Features

- **AI bots as first-class participants** — register a bot with a model and API key, add it to any room. ChatAPI handles context windowing, streaming, and delivery. Bots can also run externally and connect over REST or WebSocket like any other user.
- **LLM streaming** — token-by-token responses over WebSocket via `message.stream.start` / `message.stream.delta` / `message.stream.end`. Works with OpenAI, Anthropic, Ollama, or any OpenAI-compatible endpoint.
- **Real-time messaging** — WebSocket connections with typing indicators, presence, and at-least-once delivery guarantees.
- **JWT auth** — your backend signs JWTs with `JWT_SECRET`. ChatAPI validates them. No API keys, no sessions, no admin credentials required at runtime.
- **Durable delivery** — per-room ordered sequences with per-user ACK tracking and automatic retry for offline users.
- **Webhook** — calls your backend when a message arrives for an offline user, so you can trigger push notifications, email, or SMS.
- **Room metadata** — attach arbitrary JSON to rooms (listing IDs, order IDs, support ticket numbers, etc.).
- **Portable architecture** — repository and broker interfaces let you swap SQLite → PostgreSQL or local pub/sub → Redis without changing business logic.
- **Single binary** — no external dependencies at runtime. SQLite with WAL mode included.

## Quick Start

### Prerequisites

- Go 1.22+
- `JWT_SECRET` environment variable (required — server will not start without it)

### Run from source

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
cp .env.example .env   # set JWT_SECRET
go run ./cmd/chatapi
```

### Run with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -e ALLOWED_ORIGINS="*" \
  -v chatapi-data:/data \
  -e DATABASE_DSN="file:/data/chatapi.db" \
  hastenr/chatapi:latest
```

### Connect your first client

Your backend mints JWTs signed with `JWT_SECRET`. The `sub` claim is the user ID.

```bash
TOKEN="<your-signed-jwt>"

# Create a room
curl -X POST http://localhost:8080/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "dm", "members": ["alice", "bob"]}'

# Connect via WebSocket
# ws://localhost:8080/ws?token=<jwt>
```

## Documentation

- [Getting Started](/getting-started/) — Installation, configuration, and first API call
- [REST API](/api/rest/) — HTTP endpoint reference
- [WebSocket API](/api/websocket/) — Real-time event reference
- [TypeScript SDK](/sdk/) — `npm install @hastenr/chatapi-sdk`
- [Architecture](/architecture/) — System design and database schema
- [AI Bots](/guides/bots/) — Register bots and stream LLM responses

## Links

- [GitHub](https://github.com/hastenr/chatapi)
- [Issues](https://github.com/hastenr/chatapi/issues)
