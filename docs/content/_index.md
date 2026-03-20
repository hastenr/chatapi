+++
title = "ChatAPI Documentation"
type = "book"
weight = 1
+++

# ChatAPI

ChatAPI is a real-time messaging system that implements event-log based delivery with per-user offsets, providing at-least-once guarantees, deterministic replay, and ordered message streams per room.

## Key Features

- **Multitenant** — API key-based tenancy, per-tenant rate limiting, full data isolation
- **Real-time Messaging** — WebSocket connections for instant message delivery with typing indicators and presence
- **Durable Delivery** — Store-then-send with at-least-once guarantees, per-user ACKs, and automatic retry
- **Notifications** — Topic-based push notifications delivered over WebSocket; users subscribe to topics, your backend publishes events
- **Notification Subscriptions** — Users subscribe to topics via REST; notifications are routed to subscribers, specific users, room members, or all online users
- **Outbound Webhooks** — Offline message delivery to your backend via configurable webhook URLs
- **Room Metadata** — Attach arbitrary JSON to rooms (listing IDs, order IDs, etc.) surfaced on every room object and webhook payload
- **Room Types** — DMs, groups, and channels
- **Message Sequencing** — Per-room ordered sequences with client acknowledgment tracking
- **TypeScript SDK** — [`@hastenr/chatapi-sdk`](https://www.npmjs.com/package/@hastenr/chatapi-sdk) for browser and Node.js with built-in WebSocket reconnection
- **SQLite Backend** — WAL mode, periodic checkpointing, zero external dependencies

## Quick Start

### Prerequisites

- Go 1.22+
- `MASTER_API_KEY` environment variable (required — server will not start without it)

### Run from source

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
go mod download
go build -o bin/chatapi ./cmd/chatapi

export MASTER_API_KEY="your-secure-master-key"
export LISTEN_ADDR=":8080"
./bin/chatapi
```

### Run with Docker

```bash
docker run -d \
  -p 8080:8080 \
  -e MASTER_API_KEY=your-secure-master-key \
  -v chatapi-data:/data \
  hastenr/chatapi:latest
```

### Create your first tenant

```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Master-Key: your-secure-master-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "MyApp"}'
```

The response includes an `api_key`. **This is the only time the plaintext key is returned** — store it immediately.

## Documentation

- [Getting Started](/getting-started/) — Installation, configuration, and first steps
- [REST API](/api/rest/) — HTTP endpoints reference
- [WebSocket API](/api/websocket/) — Real-time event reference
- [API Playground](/api/playground/) — Interactive Swagger UI
- [Architecture](/architecture/) — System design and database schema

## Links

- [GitHub](https://github.com/hastenr/chatapi)
- [npm SDK](https://www.npmjs.com/package/@hastenr/chatapi-sdk)
- [Issues](https://github.com/hastenr/chatapi/issues)
- [Docker Hub](https://hub.docker.com/r/hastenr/chatapi)
