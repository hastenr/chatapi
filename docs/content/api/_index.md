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
| `GET` | `/rooms/{room_id}/members` | List room members |
| `POST` | `/rooms/{room_id}/members` | Add a member (or bot) to a room |
| `POST` | `/rooms/{room_id}/messages` | Send a message |
| `GET` | `/rooms/{room_id}/messages` | Get messages (paginated) |
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
| `typing` | Broadcast typing start/stop |

### Server → Client events

| `type` | Description |
|--------|-------------|
| `message` | New message in a room |
| `message.stream.start` | LLM bot response starting |
| `message.stream.delta` | Token chunk from LLM stream |
| `message.stream.end` | Stream complete, message persisted |
| `ack.received` | Another user acknowledged messages |
| `typing` | Another user's typing status |
| `presence.update` | User came online or went offline |
| `server.shutdown` | Server is restarting |

## SDK

The official TypeScript SDK is available on npm:

```bash
npm install @getchatapi/chatapi-sdk
```

```typescript
import { ChatAPI } from '@getchatapi/chatapi-sdk';

const client = new ChatAPI({
  baseURL: 'https://your-chatapi.com',
  token: '<your-signed-jwt>',
});

await client.connect();

client.on('message', (ev) => console.log(ev.content));
client.rooms.sendMessage('room_abc', 'Hello!');
```

## Reference

- [REST API Reference](/api/rest/) — Full endpoint documentation with examples
- [WebSocket API Reference](/api/websocket/) — Full event reference
