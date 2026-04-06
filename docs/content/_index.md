+++
title = "ChatAPI Documentation"
type = "book"
weight = 1
+++

# ChatAPI

Self-hosted, open-source chat infrastructure for AI-powered apps. Single binary, SQLite, JWT auth. Think self-hosted Sendbird or Stream Chat — but designed for apps where one or more participants is an AI.

## Key Features

- **AI Bots** — Register LLM-backed bots (OpenAI, Anthropic, or any OpenAI-compatible endpoint). ChatAPI handles context windowing, streaming, and delivery. Bots can also run externally and connect like any other user.
- **LLM Streaming** — Token-by-token streaming over WebSocket via `message.stream.start` / `message.stream.delta` / `message.stream.end` events.
- **JWT Auth** — Your backend signs JWTs with `JWT_SECRET`. ChatAPI validates them. No API keys, no sessions, no admin credentials required at runtime.
- **Real-time Messaging** — WebSocket connections with typing indicators, presence, and at-least-once delivery guarantees.
- **Message Sequencing** — Per-room ordered sequences with per-user ACK tracking and automatic retry for offline users.
- **Notifications** — Topic-based push notifications delivered over WebSocket. Subscribe via REST, publish via `POST /notify`.
- **Room Metadata** — Attach arbitrary JSON to rooms (listing IDs, order IDs, support ticket numbers, etc.).
- **Room Types** — DMs, groups, and channels.
- **Portable Architecture** — Repository and broker interfaces let you swap SQLite → PostgreSQL or local pub/sub → Redis without changing business logic.
- **Single Binary** — No external dependencies at runtime. SQLite with WAL mode included.

## Quick Start

### Prerequisites

- Go 1.22+
- `JWT_SECRET` environment variable (required — server will not start without it)

### Run from source

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
go mod download
go build -o bin/chatapi ./cmd/chatapi

export JWT_SECRET=$(openssl rand -base64 32)
export LISTEN_ADDR=":8080"
export ALLOWED_ORIGINS="*"
./bin/chatapi
```

### Run with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -e ALLOWED_ORIGINS="*" \
  -v chatapi-data:/data \
  -e DATABASE_DSN="file:/data/chatapi.db?_journal_mode=WAL&_busy_timeout=5000" \
  hastenr/chatapi:latest
```

### Connect your first client

Your backend mints JWTs signed with `JWT_SECRET`. The `sub` claim is the user ID.

```bash
# Example: sign a token with jwt-cli or any JWT library
TOKEN="<your-signed-jwt>"

# Create a room
curl -X POST http://localhost:8080/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "dm", "members": ["alice", "bob"]}'

# Connect WebSocket (browser)
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
