---
title: "WebSocket API"
weight: 22
---

# WebSocket API Reference

## Connection

```
ws://your-chatapi-instance.com/ws
wss://your-chatapi-instance.com/ws
```

## Authentication

### Browser clients

Browsers cannot set custom headers on WebSocket connections. Pass the JWT as a query parameter:

```javascript
const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${jwt}`);
```

### Server-side clients

Server clients can set the `Authorization` header during the WebSocket handshake:

```
Authorization: Bearer <jwt>
```

Example (Node.js):

```javascript
const WebSocket = require("ws");
const ws = new WebSocket("wss://your-chatapi.com/ws", {
  headers: { Authorization: "Bearer <jwt>" },
});
```

### Connection lifecycle

1. Client connects with a JWT.
2. Server validates the token and registers the connection.
3. Server broadcasts `presence.update` (online) to rooms the user belongs to.
4. Client fetches missed messages via `GET /rooms/{id}/messages?after_seq=<last_seen>` for each room.
5. On disconnect, the server waits a grace period, then broadcasts `presence.update` (offline).

### Missed messages

The server does not replay missed messages on reconnect. After reconnecting, poll each room for messages since your last known sequence number:

```http
GET /rooms/{room_id}/messages?after_seq=<last_seen_seq>
Authorization: Bearer <token>
```

---

## Message format

All frames are JSON objects with a `type` field.

---

## Client → Server messages

### send_message

Send a message to a room.

```json
{
  "type": "send_message",
  "data": {
    "room_id": "room_abc123",
    "content": "Hello!",
    "meta": "{\"mentions\":[\"bob\"]}"
  }
}
```

`meta` is optional.

### ack

Acknowledge all messages up to and including `seq`.

```json
{
  "type": "ack",
  "data": {
    "room_id": "room_abc123",
    "seq": 43
  }
}
```

### typing.start / typing.stop

```json
{"type": "typing.start", "data": {"room_id": "room_abc123"}}
{"type": "typing.stop",  "data": {"room_id": "room_abc123"}}
```

### ping

Keep-alive.

```json
{"type": "ping"}
```

---

## Server → Client events

### message

A new message was sent in a room the user belongs to.

```json
{
  "type": "message",
  "room_id": "room_abc123",
  "seq": 44,
  "message_id": "msg_def456",
  "sender_id": "alice",
  "content": "Hello!",
  "created_at": "2026-04-02T12:10:00Z"
}
```

`meta` is included only when non-empty.

### message.stream.start

An LLM bot has begun streaming a response.

```json
{
  "type": "message.stream.start",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789",
  "sender_id": "bot_support"
}
```

### message.stream.delta

A token chunk from a streaming LLM response.

```json
{
  "type": "message.stream.delta",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789",
  "delta": "Hello, how can I "
}
```

### message.stream.end

Streaming complete. The full content is included and the message has been persisted with its sequence number.

```json
{
  "type": "message.stream.end",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789",
  "sender_id": "bot_support",
  "content": "Hello, how can I help you today?",
  "seq": 45
}
```

### message.stream.error

The LLM call failed after the stream had already started. Discard any partial content accumulated for this `message_id`.

```json
{
  "type": "message.stream.error",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789"
}
```

### ack.received

Confirmation that an ACK was processed.

```json
{
  "type": "ack.received",
  "room_id": "room_abc123",
  "seq": 43,
  "user_id": "alice"
}
```

### presence.update

A user's connection status changed.

```json
{
  "type": "presence.update",
  "user_id": "bob",
  "status": "online"
}
```

`status` is `"online"` or `"offline"`.

### typing

Another user started or stopped typing.

```json
{
  "type": "typing",
  "room_id": "room_abc123",
  "user_id": "bob",
  "action": "start"
}
```

`action` is `"start"` or `"stop"`.

### error

Sent when a request is rejected. Currently emitted for rate limit violations.

```json
{
  "type": "error",
  "data": {
    "code": "rate_limited",
    "message": "too many requests"
  }
}
```

### server.shutdown

Graceful shutdown notice. Reconnect after the indicated delay.

```json
{
  "type": "server.shutdown",
  "reconnect_after_ms": 5000
}
```

---

## Reconnection

Implement exponential backoff. For browser clients, the JWT is passed as a query param and is valid for the token's `exp` claim, so refresh it if needed before reconnecting.

```javascript
let delay = 1000;

function connect() {
  const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${getJWT()}`);

  ws.onopen = () => { delay = 1000; };

  ws.onclose = () => {
    setTimeout(connect, delay);
    delay = Math.min(delay * 2, 30000);
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === "server.shutdown") {
      setTimeout(connect, msg.reconnect_after_ms);
    }
  };
}

connect();
```

## Connection limits

The server enforces a maximum of 5 concurrent WebSocket connections per user by default (configurable via `WS_MAX_CONNECTIONS_PER_USER`). Excess connections are rejected with a policy-violation close frame.
