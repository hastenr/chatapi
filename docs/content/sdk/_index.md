---
title: "TypeScript SDK"
weight: 25
---

# TypeScript SDK

The official TypeScript SDK for ChatAPI. Works in Node.js and the browser.

```bash
npm install @hastenr/chatapi-sdk
```

## Setup

Your backend mints a JWT signed with `JWT_SECRET`. Pass it to the SDK — the SDK never manages auth itself.

```typescript
import { ChatAPI } from '@hastenr/chatapi-sdk';

const client = new ChatAPI({
  baseURL: 'https://your-chatapi.com',
  token: '<your-signed-jwt>',
});

await client.connect();
```

### Config options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `baseURL` | `string` | *(required)* | ChatAPI server URL |
| `token` | `string` | *(required)* | Signed JWT (`sub` = user ID) |
| `displayName` | `string` | — | Optional display name embedded in message metadata |
| `reconnectAttempts` | `number` | `10` | Max WebSocket reconnect attempts |
| `reconnectInterval` | `number` | `1000` | Base reconnect delay in ms (exponential backoff) |
| `heartbeatInterval` | `number` | `30000` | Ping interval in ms |
| `timeout` | `number` | `10000` | REST + WebSocket connection timeout in ms |

---

## Rooms

```typescript
// Create a DM
const room = await client.rooms.create({
  type: 'dm',
  members: ['alice', 'bob'],
});

// Create a group with metadata
const support = await client.rooms.create({
  type: 'group',
  name: 'Support #1042',
  members: ['agent', 'customer'],
  metadata: JSON.stringify({ ticket_id: 't_1042' }),
});

// List rooms the current user belongs to
const rooms = await client.rooms.list();

// Get a specific room
const room = await client.rooms.get('room_abc123');

// Get members
const members = await client.rooms.getMembers('room_abc123');

// Add a member (or bot) to a room
await client.rooms.addMember('room_abc123', 'charlie');
```

---

## Messages

### Send via REST

```typescript
const msg = await client.messages.send('room_abc123', 'Hello!');
// { message_id, seq, created_at }
```

### Send via WebSocket

```typescript
// Lower latency — no HTTP round trip
client.sendMessage('room_abc123', 'Hello!');
```

### Fetch history

```typescript
// All messages
const messages = await client.messages.get('room_abc123');

// Missed messages since last seen seq
const missed = await client.messages.get('room_abc123', { after_seq: 42 });

// Paginated
const page = await client.messages.get('room_abc123', { after_seq: 0, limit: 50 });
```

### Acknowledge delivery

```typescript
await client.messages.acknowledge('room_abc123', msg.seq);
// or via WebSocket:
client.acknowledgeMessage('room_abc123', msg.seq);
```

---

## Real-time events

All events arrive via the WebSocket connection.

### Incoming messages

```typescript
client.on('message', (event) => {
  console.log(`[${event.room_id}] ${event.sender_id}: ${event.content}`);
});
```

### LLM bot streaming

```typescript
const chunks = new Map<string, string>();

client.on('message.stream.start', (event) => {
  chunks.set(event.message_id, '');
  console.log(`Bot ${event.sender_id} is typing...`);
});

client.on('message.stream.delta', (event) => {
  chunks.set(event.message_id, (chunks.get(event.message_id) ?? '') + event.delta);
  process.stdout.write(event.delta); // stream to UI
});

client.on('message.stream.end', (event) => {
  chunks.delete(event.message_id);
  console.log(`\nBot finished (seq ${event.seq}): ${event.content}`);
});
```

### Typing indicators

```typescript
// Send typing status
client.sendTyping('room_abc123', 'start');
client.sendTyping('room_abc123', 'stop');

// Receive typing events
client.on('typing', (event) => {
  console.log(`${event.user_id} is ${event.action === 'start' ? 'typing' : 'done'}`);
});
```

### Presence

```typescript
client.on('presence.update', (event) => {
  console.log(`${event.user_id} is now ${event.status}`); // 'online' | 'offline'
});
```

### Notifications

```typescript
client.on('notification', (event) => {
  const payload = JSON.parse(event.payload);
  console.log(`[${event.topic}]`, payload);
});
```

### Connection lifecycle

```typescript
client.on('connection.open', () => console.log('Connected'));
client.on('connection.lost', () => console.log('Disconnected — reconnecting...'));
client.on('connection.reconnecting', (e) => console.log(`Reconnect attempt ${e.attempt}`));
client.on('connection.failed', () => console.log('Reconnect failed after max attempts'));
```

### Remove a listener

```typescript
const handler = (event) => { /* ... */ };
client.on('message', handler);
client.off('message', handler);  // remove specific handler
client.off('message');           // remove all handlers for this event
```

---

## Notifications

```typescript
// Subscribe the current user to a topic
await client.subscriptions.subscribe('order.updates');

// List subscriptions
const subs = await client.subscriptions.list();

// Unsubscribe
await client.subscriptions.unsubscribe(subs[0].id);

// Send a notification (usually from your backend)
await client.notifications.send({
  topic: 'order.updates',
  payload: { status: 'shipped', order_id: 'ord_99' },
  targets: { topic_subscribers: true },
});
```

---

## Bots

```typescript
// Register an LLM bot
const bot = await client.bots.create({
  name: 'Support Bot',
  mode: 'llm',
  provider: 'openai',
  model: 'gpt-4o',
  api_key: 'sk-...',
  system_prompt: 'You are a helpful support agent. Be concise.',
  max_context: 20,
});

// Add the bot to a room
await client.rooms.addMember('room_abc123', bot.bot_id);

// List / get / delete
const bots = await client.bots.list();
const b = await client.bots.get(bot.bot_id);
await client.bots.delete(bot.bot_id);
```

---

## Error handling

```typescript
import {
  ChatAPIError,
  AuthenticationError,
  ValidationError,
  ConnectionError,
} from '@hastenr/chatapi-sdk';

try {
  await client.rooms.create({ type: 'dm', members: ['alice'] });
} catch (err) {
  if (err instanceof ValidationError) {
    console.error('Bad request:', err.message);
  } else if (err instanceof AuthenticationError) {
    console.error('Token expired or invalid');
  } else if (err instanceof ChatAPIError) {
    console.error(`API error [${err.code}] ${err.statusCode}: ${err.message}`);
  }
}
```

| Class | Code | Status |
|-------|------|--------|
| `AuthenticationError` | `AUTH_INVALID` | 401 |
| `ValidationError` | `VALIDATION_ERROR` | 400 |
| `ConnectionError` | `CONNECTION_FAILED` | — |
| `ChatAPIError` | varies | varies |

---

## Display names

The SDK can embed a display name in message metadata so your UI can show human-readable names without a separate user profile lookup:

```typescript
const client = new ChatAPI({ baseURL, token, displayName: 'Alice' });

// Later, on a received message:
const name = client.getSenderDisplayName(message); // 'Alice' or falls back to sender_id
```

---

## Connection management

```typescript
// Check status
client.isConnected(); // boolean

// Disconnect cleanly
await client.disconnect();

// Reconnect (e.g. after a token refresh)
client.updateConfig({ token: newJwt });
await client.connect();
```

---

## Next steps

- [REST API Reference](/api/rest/) — Use the HTTP API directly
- [WebSocket API Reference](/api/websocket/) — Full event reference
- [AI Bots Guide](/guides/bots/) — Add an LLM bot to a room
