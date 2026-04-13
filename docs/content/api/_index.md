---
title: "API Reference"
weight: 20
---

# API Reference

ChatAPI provides REST and WebSocket APIs for messaging and bots. All endpoints (except `/health` and `/metrics`) require a JWT Bearer token.

## Authentication

```
Authorization: Bearer <jwt>
```

Your backend signs JWTs with `JWT_SECRET`. The `sub` claim is the user ID for the request. There are no API keys, no master keys, and no session tokens.

**WebSocket connections** accept:
- `?token=<jwt>` query parameter — for browser clients
- `Authorization: Bearer <jwt>` header — for server-side clients

## REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/rooms` | List rooms the user belongs to |
| `POST` | `/rooms` | Create a room (DM or group) |
| `GET` | `/rooms/{room_id}` | Get room details |
| `PATCH` | `/rooms/{room_id}` | Update room name or metadata |
| `GET` | `/rooms/{room_id}/members` | List room members |
| `POST` | `/rooms/{room_id}/members` | Add a member (or bot) to a room |
| `POST` | `/rooms/{room_id}/messages` | Send a message |
| `GET` | `/rooms/{room_id}/messages` | Get messages (paginated) |
| `PUT` | `/rooms/{room_id}/messages/{message_id}` | Edit a message |
| `DELETE` | `/rooms/{room_id}/messages/{message_id}` | Delete a message |
| `POST` | `/acks` | Acknowledge message delivery |
| `POST` | `/bots` | Register a bot |
| `GET` | `/bots` | List bots |
| `GET` | `/bots/{bot_id}` | Get a bot |
| `DELETE` | `/bots/{bot_id}` | Delete a bot |
| `GET` | `/admin/dead-letters` | Failed message deliveries |
| `GET` | `/health` | Service health check (no auth) |
| `GET` | `/metrics` | Live server counters (no auth) |

## WebSocket API

Connect to `/ws` with a JWT:

```
ws://localhost:8080/ws?token=<jwt>        # browser
ws://localhost:8080/ws                    # server (Authorization header)
```

### Client → Server events

| `type` | Description |
|--------|-------------|
| `send_message` | Send a message to a room |
| `ack` | Acknowledge messages up to a sequence number |
| `typing.start` | Broadcast typing started |
| `typing.stop` | Broadcast typing stopped |
| `ping` | Keep-alive (no response) |

### Server → Client events

| `type` | Description |
|--------|-------------|
| `message` | New message in a room |
| `message.stream.start` | LLM bot response starting |
| `message.stream.delta` | Token chunk from LLM stream |
| `message.stream.end` | Stream complete, message persisted |
| `message.stream.error` | LLM call failed — discard partial content |
| `error` | Request rejected (e.g. rate limited) |
| `ack.received` | Another user acknowledged messages |
| `typing` | Another user's typing status |
| `presence.update` | User came online or went offline |
| `server.shutdown` | Server is restarting |

## Reference

- [REST API Reference](/api/rest/) — Full endpoint documentation with examples
- [WebSocket API Reference](/api/websocket/) — Full event reference
