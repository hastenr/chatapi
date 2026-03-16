---
title: "WebSocket API"
weight: 22
---

# WebSocket API Reference

The ChatAPI WebSocket API enables real-time, bidirectional communication for instant messaging and presence updates.

## Connection

### Endpoint

```
ws://your-chatapi-instance.com/ws
wss://your-chatapi-instance.com/ws  # for HTTPS
```

### Authentication

WebSocket authentication depends on the type of client:

#### Server / Node.js clients

Server-side clients can set custom HTTP headers during the WebSocket handshake. Pass credentials via headers:

```
X-API-Key: your-tenant-api-key
X-User-Id: user-identifier
```

Alternatively, `user_id` can be passed as a query parameter alongside the `X-API-Key` header:

```
/ws?user_id=user-identifier
```

#### Browser clients

Browsers do not allow setting custom headers on WebSocket connections. Use the token-based flow instead:

**Step 1 — Fetch a token** via `POST /ws/token` (standard REST call with `X-API-Key` + `X-User-Id` headers):

```javascript
const resp = await fetch('/ws/token', {
  method: 'POST',
  headers: {
    'X-API-Key': apiKey,
    'X-User-Id': userId
  }
});
const { token } = await resp.json();
```

**Step 2 — Connect using the token** as a query parameter:

```javascript
const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${token}`);
```

Tokens are **single-use** and expire after **60 seconds**. If the connection attempt fails, request a new token before retrying.

> **Security note**: The `?api_key=` query parameter auth method was removed as a security fix — API keys passed in query parameters appear in server logs and proxy access logs. Do not attempt to authenticate using `?api_key=`.

### Connection Lifecycle

1. **Connect**: Client establishes WebSocket connection
2. **Authenticate**: Server validates credentials (header or token)
3. **Register**: Connection registered for real-time events
4. **Sync**: Server streams missed messages
5. **Communicate**: Bidirectional message exchange
6. **Disconnect**: Clean connection termination

## Message Protocol

All WebSocket messages use JSON format:

```json
{
  "type": "message_type",
  "data": { ... }
}
```

## Client → Server Messages

### Send Message

Send a message to a room.

```json
{
  "type": "send_message",
  "room_id": "room_abc123",
  "content": "Hello, world!",
  "meta": "{\"type\":\"text\",\"mentions\":[\"user2\"]}"
}
```

### Acknowledge Messages

Mark messages as delivered.

```json
{
  "type": "ack",
  "room_id": "room_abc123",
  "seq": 43
}
```

### Typing Indicators

Send typing start/stop events.

```json
{
  "type": "typing.start",
  "room_id": "room_abc123"
}
```

```json
{
  "type": "typing.stop",
  "room_id": "room_abc123"
}
```

### Subscribe to Room

Explicitly subscribe to room events (optional - automatic for room members).

```json
{
  "type": "subscribe",
  "room_id": "room_abc123"
}
```

## Server → Client Messages

### Message Delivery

New messages in subscribed rooms.

```json
{
  "type": "message",
  "room_id": "room_abc123",
  "message_id": "msg_def456",
  "seq": 44,
  "sender_id": "user1",
  "content": "Hello, world!",
  "meta": "{\"type\":\"text\",\"mentions\":[\"user2\"]}",
  "created_at": "2025-12-13T12:10:00Z"
}
```

### Acknowledgment Received

Confirmation that an ACK was processed.

```json
{
  "type": "ack.received",
  "room_id": "room_abc123",
  "seq": 43,
  "user_id": "user2"
}
```

### Presence Updates

User online/offline status changes.

```json
{
  "type": "presence.update",
  "user_id": "user2",
  "status": "online"
}
```

```json
{
  "type": "presence.update",
  "user_id": "user2",
  "status": "offline"
}
```

### Typing Indicators

Other users' typing status.

```json
{
  "type": "typing",
  "room_id": "room_abc123",
  "user_id": "user2",
  "action": "start"
}
```

```json
{
  "type": "typing",
  "room_id": "room_abc123",
  "user_id": "user2",
  "action": "stop"
}
```

### Notifications

Real-time notification delivery.

```json
{
  "type": "notification",
  "notification_id": "notif_ghi789",
  "topic": "order.shipped",
  "payload": {
    "order_id": "12345",
    "tracking_number": "1Z999AA1234567890"
  }
}
```

### Server Shutdown

Graceful shutdown notification.

```json
{
  "type": "server.shutdown",
  "reconnect_after_ms": 5000
}
```

## Connection Behavior

### On Connect

1. **Validation**: Server validates credentials (API key + user ID, or token)
2. **Registration**: Connection registered for the user
3. **Presence Broadcast**: Online status sent to room members
4. **Message Sync**: Missed messages streamed in order
5. **Subscription**: Automatic subscription to user's rooms

### On Disconnect

1. **Presence Update**: Offline status broadcast (after 5s grace period)
2. **Cleanup**: Connection removed from active connections
3. **Persistence**: User state maintained for reconnection

### Reconnection

Clients should implement exponential backoff reconnection.

**Node.js / server clients:**

```javascript
let reconnectDelay = 1000;

function connect() {
  const ws = new WebSocket('ws://your-chatapi.com/ws', [], {
    headers: {
      'X-API-Key': apiKey,
      'X-User-Id': userId
    }
  });

  ws.onclose = () => {
    setTimeout(connect, reconnectDelay);
    reconnectDelay = Math.min(reconnectDelay * 2, 30000);
  };

  ws.onopen = () => {
    reconnectDelay = 1000;
  };
}
```

**Browser clients** must refresh the token before each reconnect attempt:

```javascript
let reconnectDelay = 1000;

async function getToken() {
  const resp = await fetch('/ws/token', {
    method: 'POST',
    headers: { 'X-API-Key': apiKey, 'X-User-Id': userId }
  });
  const { token } = await resp.json();
  return token;
}

async function connect() {
  const token = await getToken();
  const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${token}`);

  ws.onclose = () => {
    setTimeout(connect, reconnectDelay);
    reconnectDelay = Math.min(reconnectDelay * 2, 30000);
  };

  ws.onopen = () => {
    reconnectDelay = 1000;
  };
}
```

## Message Ordering

Messages are delivered in strict sequence order per room:

- **Sequence Numbers**: Each message has a unique `seq` number
- **Guaranteed Order**: Messages delivered in `seq` order
- **No Gaps**: Clients may receive messages with non-consecutive seq numbers
- **ACKs**: Clients acknowledge the highest contiguous seq number received

## Error Handling

### Connection Errors

- **Invalid Credentials**: Connection closed immediately
- **Rate Limiting**: Connection temporarily suspended
- **Server Errors**: Connection closed with error message

### Message Errors

Invalid messages result in error responses:

```json
{
  "type": "error",
  "code": "VALIDATION_ERROR",
  "message": "Invalid message format",
  "request_id": "req_123"
}
```

## Heartbeat/Ping

WebSocket connections include automatic ping/pong:

- **Ping Interval**: 30 seconds
- **Timeout**: 60 seconds of inactivity
- **Automatic**: Handled by WebSocket protocol

## Client Implementation Examples

### JavaScript (Browser — token flow)

```javascript
class ChatAPIClient {
  constructor(apiKey, userId, baseUrl = 'https://your-chatapi.com') {
    this.apiKey = apiKey;
    this.userId = userId;
    this.baseUrl = baseUrl;
    this.ws = null;
    this.reconnectDelay = 1000;
  }

  async fetchToken() {
    const resp = await fetch(`${this.baseUrl}/ws/token`, {
      method: 'POST',
      headers: {
        'X-API-Key': this.apiKey,
        'X-User-Id': this.userId
      }
    });
    if (!resp.ok) throw new Error('Failed to fetch WS token');
    const { token } = await resp.json();
    return token;
  }

  async connect() {
    const token = await this.fetchToken();
    const wsUrl = this.baseUrl.replace(/^http/, 'ws');
    this.ws = new WebSocket(`${wsUrl}/ws?token=${token}`);

    this.ws.onopen = () => {
      console.log('Connected to ChatAPI');
      this.reconnectDelay = 1000;
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onclose = () => {
      console.log('Disconnected, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectDelay);
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  sendMessage(roomId, content) {
    this.send({
      type: 'send_message',
      room_id: roomId,
      content: content
    });
  }

  acknowledge(roomId, seq) {
    this.send({
      type: 'ack',
      room_id: roomId,
      seq: seq
    });
  }

  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  handleMessage(message) {
    switch (message.type) {
      case 'message':
        console.log('New message:', message);
        break;
      case 'presence.update':
        console.log('Presence update:', message);
        break;
      case 'server.shutdown':
        console.log('Server shutting down, reconnecting in', message.reconnect_after_ms, 'ms');
        setTimeout(() => this.connect(), message.reconnect_after_ms);
        break;
    }
  }
}

// Usage
const client = new ChatAPIClient('your-api-key', 'user123');
client.connect();
```

### Python (websockets library — server client)

```python
import asyncio
import json
import websockets
from websockets.exceptions import ConnectionClosedError

class ChatAPIClient:
    def __init__(self, api_key, user_id, url="ws://localhost:8080/ws"):
        self.api_key = api_key
        self.user_id = user_id
        self.url = url
        self.websocket = None
        self.reconnect_delay = 1.0

    async def connect(self):
        headers = {
            "X-API-Key": self.api_key,
            "X-User-Id": self.user_id
        }

        try:
            self.websocket = await websockets.connect(
                self.url,
                extra_headers=headers
            )
            print("Connected to ChatAPI")
            self.reconnect_delay = 1.0

            asyncio.create_task(self.handle_messages())

        except Exception as e:
            print(f"Connection failed: {e}")
            await self.reconnect()

    async def reconnect(self):
        await asyncio.sleep(self.reconnect_delay)
        self.reconnect_delay = min(self.reconnect_delay * 2, 30.0)
        await self.connect()

    async def handle_messages(self):
        try:
            async for message in self.websocket:
                data = json.loads(message)
                await self.handle_message(data)
        except ConnectionClosedError:
            print("Connection closed, reconnecting...")
            await self.reconnect()

    async def handle_message(self, message):
        msg_type = message.get('type')

        if msg_type == 'message':
            print(f"New message in {message['room_id']}: {message['content']}")
        elif msg_type == 'presence.update':
            print(f"User {message['user_id']} is {message['status']}")
        elif msg_type == 'server.shutdown':
            delay = message.get('reconnect_after_ms', 5000) / 1000
            print(f"Server shutting down, reconnecting in {delay}s")
            await asyncio.sleep(delay)
            await self.connect()

    async def send_message(self, room_id, content):
        message = {
            "type": "send_message",
            "room_id": room_id,
            "content": content
        }
        await self.send(message)

    async def acknowledge(self, room_id, seq):
        message = {
            "type": "ack",
            "room_id": room_id,
            "seq": seq
        }
        await self.send(message)

    async def send(self, data):
        if self.websocket:
            await self.websocket.send(json.dumps(data))

# Usage
async def main():
    client = ChatAPIClient('your-api-key', 'user123')
    await client.connect()

    while True:
        await asyncio.sleep(1)

asyncio.run(main())
```

### Go (gorilla/websocket — server client)

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "net/url"
    "time"

    "github.com/gorilla/websocket"
)

type ChatAPIClient struct {
    apiKey         string
    userId         string
    serverURL      string
    conn           *websocket.Conn
    reconnectDelay time.Duration
}

func NewChatAPIClient(apiKey, userId, serverURL string) *ChatAPIClient {
    return &ChatAPIClient{
        apiKey:         apiKey,
        userId:         userId,
        serverURL:      serverURL,
        reconnectDelay: time.Second,
    }
}

func (c *ChatAPIClient) connect() error {
    u, err := url.Parse(c.serverURL + "/ws")
    if err != nil {
        return err
    }

    // Server clients authenticate via headers
    headers := http.Header{}
    headers.Set("X-API-Key", c.apiKey)
    headers.Set("X-User-Id", c.userId)

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
    if err != nil {
        return err
    }

    c.conn = conn
    c.reconnectDelay = time.Second
    log.Println("Connected to ChatAPI")

    go c.handleMessages()
    return nil
}

func (c *ChatAPIClient) handleMessages() {
    defer c.conn.Close()

    for {
        var message map[string]interface{}
        err := c.conn.ReadJSON(&message)
        if err != nil {
            log.Println("Read error:", err)
            c.reconnect()
            return
        }

        c.handleMessage(message)
    }
}

func (c *ChatAPIClient) handleMessage(message map[string]interface{}) {
    msgType, ok := message["type"].(string)
    if !ok {
        return
    }

    switch msgType {
    case "message":
        log.Printf("New message in %s: %s", message["room_id"], message["content"])
    case "presence.update":
        log.Printf("User %s is %s", message["user_id"], message["status"])
    case "server.shutdown":
        if delay, ok := message["reconnect_after_ms"].(float64); ok {
            log.Printf("Server shutting down, reconnecting in %.0fms", delay)
            time.Sleep(time.Duration(delay) * time.Millisecond)
            c.connect()
        }
    }
}

func (c *ChatAPIClient) reconnect() {
    c.conn = nil
    time.Sleep(c.reconnectDelay)
    c.reconnectDelay = min(c.reconnectDelay*2, 30*time.Second)

    if err := c.connect(); err != nil {
        log.Println("Reconnect failed:", err)
        c.reconnect()
    }
}

func (c *ChatAPIClient) SendMessage(roomID, content string) error {
    message := map[string]interface{}{
        "type":    "send_message",
        "room_id": roomID,
        "content": content,
    }
    return c.send(message)
}

func (c *ChatAPIClient) Acknowledge(roomID string, seq int) error {
    message := map[string]interface{}{
        "type":    "ack",
        "room_id": roomID,
        "seq":     seq,
    }
    return c.send(message)
}

func (c *ChatAPIClient) send(data interface{}) error {
    if c.conn == nil {
        return errors.New("not connected")
    }
    return c.conn.WriteJSON(data)
}

func min(a, b time.Duration) time.Duration {
    if a < b {
        return a
    }
    return b
}

// Usage
func main() {
    client := NewChatAPIClient("your-api-key", "user123", "ws://localhost:8080")
    if err := client.connect(); err != nil {
        log.Fatal("Failed to connect:", err)
    }

    select {}
}
```

## Best Practices

### Connection Management

- **Single Connection**: Maintain one WebSocket connection per user
- **Reconnection Logic**: Implement exponential backoff
- **Graceful Shutdown**: Handle server shutdown messages
- **Connection Pooling**: Avoid multiple connections for the same user (server enforces `MAX_CONNECTIONS_PER_USER`, default 5)

### Message Handling

- **Deduplication**: Track message IDs to prevent duplicates
- **Sequence Tracking**: Maintain per-room sequence numbers
- **ACK Batching**: Send ACKs for highest contiguous sequence
- **Error Recovery**: Handle network interruptions gracefully

### Performance

- **Message Batching**: Send multiple messages in single WebSocket frame when possible
- **Compression**: Enable WebSocket compression for large messages
- **Ping/Pong**: Monitor connection health
- **Resource Limits**: Implement message size and rate limits

### Security

- **Authentication**: Always use secure WebSocket (WSS) in production
- **Token Flow**: Use `POST /ws/token` for browser clients — never expose API keys in URLs
- **Input Validation**: Validate all incoming messages
- **Rate Limiting**: Respect server rate limits
- **CORS**: Configure `WS_ALLOWED_ORIGINS` on the server to restrict which browser origins may connect
