---
title: "REST API"
weight: 21
---

# REST API Reference

## Authentication

ChatAPI never issues tokens — your backend mints them and your client uses them. The flow is:

```
1. Client logs in to your backend (your existing auth)
2. Your backend mints a JWT signed with JWT_SECRET and returns it to the client
3. Client passes the token to ChatAPI on every request
```

The `JWT_SECRET` lives only on your backend and on the ChatAPI server — it is never exposed to the client.

Mint a token on your backend:

```javascript
// Node.js
import jwt from 'jsonwebtoken';

const token = jwt.sign(
  { sub: 'user_alice' },
  process.env.JWT_SECRET,
  { expiresIn: '24h' }
);
// return this token to your client
```

```go
// Go
claims := jwt.MapClaims{
    "sub": "user_alice",
    "exp": time.Now().Add(24 * time.Hour).Unix(),
}
token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
    SignedString([]byte(os.Getenv("JWT_SECRET")))
// return this token to your client
```

Pass the token on every ChatAPI request:

```
Authorization: Bearer <token>          # REST
ws://localhost:8080/ws?token=<token>   # WebSocket (browser)
```

Error responses use a flat JSON shape:

```json
{"error": "not_found", "message": "room not found"}
```

Common status codes: `400` invalid request, `401` missing/invalid token, `403` forbidden, `404` not found, `500` server error.

---

## Health & Metrics

### Health check

```http
GET /health
```

No authentication required.

```json
{"status": "ok", "db_writable": true}
```

### Server metrics

```http
GET /metrics
```

No authentication required. Live counters since server start.

```json
{
  "active_connections": 42,
  "messages_sent": 18340,
  "broadcast_drops": 3,
  "delivery_attempts": 21005,
  "delivery_failures": 12,
  "uptime_seconds": 86400
}
```

---

## Rooms

### List rooms

Returns all rooms the authenticated user belongs to.

```http
GET /rooms
Authorization: Bearer <token>
```

```json
{
  "rooms": [
    {
      "room_id": "room_abc123",
      "type": "dm",
      "name": null,
      "metadata": "{\"listing_id\":\"lst_99\"}",
      "last_seq": 42,
      "created_at": "2026-04-02T12:00:00Z"
    }
  ]
}
```

### Create room

```http
POST /rooms
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "type": "dm",
  "members": ["alice", "bob"],
  "name": "Optional name",
  "metadata": "{\"order_id\":\"ord_42\"}"
}
```

- `type`: `"dm"` | `"group"` — required
- `members`: array of user IDs — required
- `name`: optional display name
- `metadata`: optional arbitrary JSON string for app-level context (listing IDs, order IDs, etc.)

DMs are deduplicated — creating a DM between the same two users always returns the same room.

Returns `200` with the created room object.

### Get room

```http
GET /rooms/{room_id}
Authorization: Bearer <token>
```

Returns the room object. `403` if the user is not a member.

### Update room

```http
PATCH /rooms/{room_id}
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "name": "New name",
  "metadata": "{\"order_id\":\"ord_99\"}"
}
```

Both fields are optional. Omitting a field leaves it unchanged. Returns the updated room object.

---

## Members

### List members

```http
GET /rooms/{room_id}/members
Authorization: Bearer <token>
```

```json
{
  "members": [
    {"user_id": "alice", "role": "member", "joined_at": "2026-04-02T12:00:00Z"},
    {"user_id": "bob",   "role": "member", "joined_at": "2026-04-02T12:00:00Z"}
  ]
}
```

### Add member

Adds a user or bot to a room.

```http
POST /rooms/{room_id}/members
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{"user_id": "charlie"}
```

Returns `200` on success.

---

## Messages

### Get messages

```http
GET /rooms/{room_id}/messages?after_seq=40&limit=50
Authorization: Bearer <token>
```

Query parameters:
- `after_seq` — return messages with `seq > after_seq` (optional, default 0)
- `limit` — max messages to return (optional, default 50, max 100)

```json
{
  "messages": [
    {
      "message_id": "msg_abc123",
      "chatroom_id": "room_abc123",
      "sender_id": "alice",
      "seq": 41,
      "content": "Hello!",
      "meta": "",
      "created_at": "2026-04-02T12:10:00Z"
    }
  ]
}
```

### Send message

```http
POST /rooms/{room_id}/messages
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "content": "Hello!",
  "meta": "{\"mentions\":[\"bob\"]}"
}
```

- `content`: message text — required
- `meta`: optional arbitrary JSON string (mentions, reactions, etc.)

Returns `200` with the created message object.

### Edit message

Only the original sender may edit a message.

```http
PUT /rooms/{room_id}/messages/{message_id}
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{"content": "Updated content"}
```

Returns the updated message object. `403` if not the sender.

### Delete message

Only the original sender may delete a message.

```http
DELETE /rooms/{room_id}/messages/{message_id}
Authorization: Bearer <token>
```

Returns `204 No Content`. `403` if not the sender.

---

## Acknowledgments

Mark messages as delivered up to a given sequence number.

```http
POST /acks
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{"room_id": "room_abc123", "seq": 43}
```

Returns `200 OK`.

---

## Bots

### Register a bot

```http
POST /bots
Authorization: Bearer <token>
Content-Type: application/json
```

```json
{
  "name": "Support Bot",
  "llm_base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
  "llm_api_key_env": "GEMINI_API_KEY",
  "model": "gemini-2.0-flash"
}
```

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Display name |
| `llm_base_url` | Yes | OpenAI-compatible base URL |
| `llm_api_key_env` | Yes | Name of the server env var holding the API key |
| `model` | Yes | Model identifier (e.g. `gemini-2.0-flash`, `gpt-4o`) |

Returns `201` with the created bot object:

```json
{
  "bot_id": "bot_abc123",
  "name": "Support Bot",
  "llm_base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
  "llm_api_key_env": "GEMINI_API_KEY",
  "model": "gemini-2.0-flash",
  "created_at": "2026-04-02T12:00:00Z"
}
```

### List bots

```http
GET /bots
Authorization: Bearer <token>
```

### Get bot

```http
GET /bots/{bot_id}
Authorization: Bearer <token>
```

### Delete bot

```http
DELETE /bots/{bot_id}
Authorization: Bearer <token>
```

Returns `204 No Content`.

---

## Admin

### Dead letters

Returns undelivered messages that have exceeded the retry limit (5 attempts).

```http
GET /admin/dead-letters?limit=100
Authorization: Bearer <token>
```

```json
{
  "failed_messages": [
    {
      "id": 1,
      "user_id": "alice",
      "chatroom_id": "room_abc123",
      "message_id": "msg_abc123",
      "seq": 41,
      "attempts": 5,
      "created_at": "2026-04-02T12:10:00Z"
    }
  ]
}
```

---

## Content types and formats

- All request bodies: `application/json` (UTF-8)
- All timestamps: ISO 8601 (`2026-04-02T12:00:00Z`)
