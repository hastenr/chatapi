---
title: "Guides"
weight: 30
---

# Guides

Practical guides for integrating ChatAPI into your applications.

## Available Guides

### [Tenants](/guides/tenants/)
Create and manage tenants, configure API keys, and set up multi-tenant environments.

## Quick Reference

The fastest path to a working integration:

**1. Create a tenant**
```bash
curl -X POST http://localhost:8080/admin/tenants \
  -H "X-Master-Key: $MASTER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "MyApp"}'
```

**2. Create a room**
```bash
curl -X POST http://localhost:8080/rooms \
  -H "X-API-Key: $API_KEY" \
  -H "X-User-Id: user1" \
  -H "Content-Type: application/json" \
  -d '{"type":"dm","members":["user1","user2"]}'
```

**3. Connect via WebSocket (Node.js)**
```typescript
import { ChatAPI } from '@hastenr/chatapi-sdk';

const client = new ChatAPI({ baseURL, apiKey, userId: 'user1' });
await client.connect();
client.on('message', (ev) => console.log(ev.content));
client.sendMessage('room_abc', 'Hello!');
```

**4. Subscribe to notifications**
```typescript
await client.subscriptions.subscribe('order.updates');
client.on('notification', (ev) => {
  const payload = JSON.parse(ev.payload);
  console.log(payload.message);
});
```

**5. Send a notification from your backend**
```bash
curl -X POST http://localhost:8080/notify \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "order.updates",
    "payload": { "status": "shipped", "message": "Your order is on its way!" },
    "targets": { "topic_subscribers": true }
  }'
```

## See Also

- [REST API Reference](/api/rest/) — Full endpoint documentation
- [WebSocket API Reference](/api/websocket/) — Real-time event reference
- [Getting Started](/getting-started/) — Installation and configuration
- [GitHub Issues](https://github.com/hastenr/chatapi/issues) — Report problems or request features
