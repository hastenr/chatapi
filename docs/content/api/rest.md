---
title: "REST API"
weight: 21
---

# REST API Reference

The ChatAPI REST API provides HTTP endpoints for chat operations. All endpoints require authentication and return JSON responses.

## Authentication

### Standard Endpoints

Include these headers with every request:

```
X-API-Key: your-tenant-api-key
X-User-Id: user-identifier
```

- **X-API-Key**: Identifies your tenant (organization). API keys are stored as SHA-256 hashes in the database; the plaintext key is returned only once at tenant creation and cannot be retrieved again.
- **X-User-Id**: Identifies the user performing the action

### Admin Endpoints

Admin endpoints require master API key authentication:

```
X-Master-Key: your-master-api-key
```

- **X-Master-Key**: Master key for administrative operations (set via `MASTER_API_KEY` env var)
- Used for tenant creation and system administration

## Rooms

### List Rooms

List all rooms the authenticated user belongs to.

```http
GET /rooms
```

**Response:**
```json
{
  "rooms": [
    {
      "room_id": "room_abc123",
      "type": "dm",
      "unique_key": "dm:alice:bob",
      "name": null,
      "metadata": "{\"listing_id\":\"lst_99\"}",
      "last_seq": 42,
      "created_at": "2025-12-13T12:00:00Z"
    }
  ]
}
```

**Status Codes:**
- `200` - Success
- `401` - Authentication failed

### Create Room

Create a new chat room (DM, group, or channel).

```http
POST /rooms
```

**Request Body:**
```json
{
  "type": "dm" | "group" | "channel",
  "members": ["user1", "user2"],
  "name": "Optional room name",
  "metadata": "{\"listing_id\":\"lst_99\",\"order_id\":\"ord_42\"}"
}
```

The `metadata` field is an optional arbitrary JSON string. It is stored as-is and returned on every room object. Use it to attach app-level context (listing IDs, order IDs, etc.) that your application needs without storing it separately. It is also included in offline webhook payloads.

**Response:**
```json
{
  "room_id": "room_abc123",
  "type": "dm",
  "unique_key": "dm:user1:user2",
  "name": null,
  "metadata": "{\"listing_id\":\"lst_99\",\"order_id\":\"ord_42\"}",
  "last_seq": 0,
  "created_at": "2025-12-13T12:00:00Z"
}
```

**Status Codes:**
- `201` - Room created successfully
- `400` - Invalid request parameters
- `401` - Authentication failed
- `409` - Room already exists (for DMs)

### Get Room

Retrieve room information.

```http
GET /rooms/{room_id}
```

**Response:**
```json
{
  "room_id": "room_abc123",
  "type": "group",
  "name": "Team Chat",
  "metadata": null,
  "last_seq": 42,
  "created_at": "2025-12-13T12:00:00Z"
}
```

### List Room Members

Get all members of a room.

```http
GET /rooms/{room_id}/members
```

**Response:**
```json
[
  {
    "user_id": "user1",
    "role": "admin",
    "joined_at": "2025-12-13T12:00:00Z"
  },
  {
    "user_id": "user2",
    "role": "member",
    "joined_at": "2025-12-13T12:05:00Z"
  }
]
```

## Messages

### Send Message

Send a message to a room.

```http
POST /rooms/{room_id}/messages
```

**Request Body:**
```json
{
  "content": "Hello, world!",
  "meta": "{\"type\":\"text\",\"mentions\":[\"user2\"]}"
}
```

The `meta` field is an optional arbitrary JSON string that is stored with the message and returned when reading messages. Use it for client-defined message metadata such as type hints, mention lists, or reaction data.

**Response:**
```json
{
  "message_id": "msg_abc123",
  "seq": 43,
  "created_at": "2025-12-13T12:10:00Z"
}
```

### Get Messages

Retrieve messages from a room.

```http
GET /rooms/{room_id}/messages?after_seq=40&limit=50
```

**Query Parameters:**
- `after_seq` (optional): Get messages after this sequence number
- `limit` (optional): Maximum messages to return (default: 50, max: 100)

**Response:**
```json
[
  {
    "message_id": "msg_abc123",
    "sender_id": "user1",
    "seq": 41,
    "content": "Hello, world!",
    "meta": "{\"type\":\"text\",\"mentions\":[\"user2\"]}",
    "created_at": "2025-12-13T12:10:00Z"
  }
]
```

## Delivery & Acknowledgments

### Acknowledge Messages

Mark messages as delivered up to a specific sequence number.

```http
POST /acks
```

**Request Body:**
```json
{
  "room_id": "room_abc123",
  "seq": 43
}
```

**Response:**
```json
{
  "success": true
}
```

## Notifications

### Send Notification

Send a notification to users or room members.

```http
POST /notify
```

**Request Body:**
```json
{
  "topic": "order.shipped",
  "payload": {
    "order_id": "12345",
    "tracking_number": "1Z999AA1234567890"
  },
  "targets": {
    "user_ids": ["user1", "user2"],
    "room_id": "room_abc123",
    "topic_subscribers": true
  }
}
```

**Response:**
```json
{
  "notification_id": "notif_abc123",
  "created_at": "2025-12-13T12:15:00Z"
}
```

## WebSocket Tokens

### Issue WebSocket Token

Issue a short-lived, single-use authentication token for browser WebSocket connections. Browser clients cannot set custom HTTP headers on WebSocket connections, so this token-based flow is required.

```http
POST /ws/token
```

**Authentication:** Standard `X-API-Key` + `X-User-Id` headers.

**Response:**
```json
{
  "token": "wst_a1b2c3d4e5f6...",
  "expires_in": 60
}
```

After obtaining a token, connect to the WebSocket endpoint:

```
wss://your-chatapi.com/ws?token=wst_a1b2c3d4e5f6...
```

The token is valid for **60 seconds** and is consumed on first use. If the connection attempt fails, request a new token before retrying.

**Status Codes:**
- `200` - Token issued successfully
- `401` - Invalid API key or user ID

## Health & Monitoring

### Health Check

Check service health and status.

```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "uptime": "2h30m45s",
  "db_writable": true
}
```

### Server Metrics

Returns live server counters. No authentication required.

```http
GET /metrics
```

**Response:**
```json
{
  "active_connections": 42,
  "messages_sent": 18340,
  "delivery_attempts": 21005,
  "delivery_failures": 12,
  "dropped_broadcasts": 3,
  "uptime_seconds": 86400
}
```

**Fields:**
- `active_connections` — currently open WebSocket connections
- `messages_sent` — total messages sent since server start
- `delivery_attempts` — total background delivery attempts
- `delivery_failures` — deliveries that failed permanently (see dead-letter queue)
- `dropped_broadcasts` — broadcasts dropped due to slow consumers
- `uptime_seconds` — server uptime in seconds

## Admin Endpoints

### Create Tenant

Create a new tenant with an auto-generated API key. Requires `X-Master-Key` authentication (not `X-API-Key`).

```http
POST /admin/tenants
```

**Authentication:**
```
X-Master-Key: your-master-api-key
```

**Request Body:**
```json
{
  "name": "MyCompany"
}
```

**Response:**
```json
{
  "tenant_id": "tenant_abc123",
  "name": "MyCompany",
  "api_key": "sk_abc123def456ghi789jkl012mno345pqr678stu901vwx",
  "created_at": "2025-12-13T12:00:00Z"
}
```

**Status Codes:**
- `201` - Tenant created successfully
- `400` - Invalid request parameters
- `401` - Invalid master API key
- `500` - Server error

**Security Note:** The `api_key` is returned **only in this response**. It is stored as a SHA-256 hash in the database — the plaintext can never be retrieved again. Copy it immediately and store it securely (e.g. in a secrets manager).

### Dead Letters (Admin)

View failed deliveries (admin endpoint).

```http
GET /admin/dead-letters?tenant_id=tenant_abc&limit=100
```

**Authentication:**
```
X-Master-Key: your-master-api-key
```

**Query Parameters:**
- `tenant_id` (required): Tenant identifier
- `limit` (optional): Maximum results (default: 50, max: 1000)

**Response:**
```json
{
  "failed_messages": [
    {
      "message_id": "msg_abc123",
      "user_id": "user1",
      "room_id": "room_abc123",
      "attempts": 5,
      "last_attempt_at": "2025-12-13T12:20:00Z",
      "error": "connection timeout"
    }
  ],
  "failed_notifications": [
    {
      "notification_id": "notif_abc123",
      "topic": "order.shipped",
      "attempts": 3,
      "last_attempt_at": "2025-12-13T12:25:00Z",
      "error": "user offline"
    }
  ]
}
```

## Error Responses

All endpoints may return these error formats:

**Validation Error:**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid room type. Must be one of: dm, group, channel",
    "field": "type"
  }
}
```

**Authentication Error:**
```json
{
  "success": false,
  "error": {
    "code": "AUTHENTICATION_ERROR",
    "message": "Invalid API key"
  }
}
```

**Rate Limit Exceeded:**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Try again in 60 seconds"
  }
}
```

## Rate Limiting

ChatAPI implements per-tenant rate limiting:

- **Default Limit**: 100 requests per second per tenant
- **Response Headers**:
  ```
  X-RateLimit-Limit: 100
  X-RateLimit-Remaining: 95
  X-RateLimit-Reset: 1640995200
  ```
- **429 Status**: Returned when limits are exceeded
- **Retry-After Header**: Indicates when to retry

## Content Types

- **Request Body**: `application/json`
- **Response Body**: `application/json`
- **Character Encoding**: UTF-8

## Time Formats

All timestamps use ISO 8601 format:

```
2025-12-13T12:00:00Z
2025-12-13T12:00:00.123456Z
```

## Pagination

Endpoints that return lists support pagination:

- `limit`: Maximum items per page (default: 50, max: 100)
- `offset`: Number of items to skip (default: 0)

## Versioning

API versioning is handled via URL paths:

- Current version: No prefix (v1 implied)
- Future versions: `/v2/`, `/v3/`, etc.

## SDK Examples

### JavaScript (Fetch API)

```javascript
const API_KEY = 'your-api-key';
const USER_ID = 'user123';

async function sendMessage(roomId, content) {
  const response = await fetch(`/rooms/${roomId}/messages`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      'X-User-Id': USER_ID
    },
    body: JSON.stringify({ content })
  });

  return response.json();
}
```

### Python (requests)

```python
import requests

API_KEY = 'your-api-key'
USER_ID = 'user123'
BASE_URL = 'https://your-chatapi.com'

def send_message(room_id, content):
    headers = {
        'Content-Type': 'application/json',
        'X-API-Key': API_KEY,
        'X-User-Id': USER_ID
    }

    data = {'content': content}

    response = requests.post(
        f'{BASE_URL}/rooms/{room_id}/messages',
        json=data,
        headers=headers
    )

    return response.json()
```

### Go (net/http)

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const (
    apiKey  = "your-api-key"
    userID  = "user123"
    baseURL = "https://your-chatapi.com"
)

func sendMessage(roomID, content string) error {
    data := map[string]string{"content": content}
    jsonData, _ := json.Marshal(data)

    req, _ := http.NewRequest("POST",
        baseURL+"/rooms/"+roomID+"/messages",
        bytes.NewBuffer(jsonData))

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", apiKey)
    req.Header.Set("X-User-Id", userID)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```
