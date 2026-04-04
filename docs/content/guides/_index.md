---
title: "Guides"
weight: 30
---

# Guides

Practical guides for integrating ChatAPI into your applications.

## Available Guides

### [AI Bots](/guides/bots/)
Register LLM-backed bots, add them to rooms, and stream responses to users in real time. Works with OpenAI, Anthropic, Ollama, and any OpenAI-compatible endpoint.

## Quick Reference

The fastest path to a working integration:

**1. Start the server**
```bash
export JWT_SECRET=$(openssl rand -base64 32)
export ALLOWED_ORIGINS="*"
./bin/chatapi
```

**2. Mint a JWT for a user** (your backend does this)
```go
claims := jwt.MapClaims{"sub": "user1", "exp": time.Now().Add(24 * time.Hour).Unix()}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signed, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
```

**3. Create a room**
```bash
curl -X POST http://localhost:8080/rooms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type":"dm","members":["user1","user2"]}'
```

**4. Connect via WebSocket (browser)**
```javascript
const ws = new WebSocket(`ws://localhost:8080/ws?token=${jwt}`);
ws.onmessage = (event) => console.log(JSON.parse(event.data));
```

**5. Subscribe to notifications**
```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic": "order.updates"}'
```

**6. Send a notification from your backend**
```bash
curl -X POST http://localhost:8080/notify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "order.updates",
    "payload": {"status": "shipped", "message": "Your order is on its way!"},
    "targets": {"topic_subscribers": true}
  }'
```

## See Also

- [REST API Reference](/api/rest/) — Full endpoint documentation
- [WebSocket API Reference](/api/websocket/) — Real-time event reference
- [Getting Started](/getting-started/) — Installation and configuration
- [GitHub Issues](https://github.com/hastenr/chatapi/issues) — Report problems or request features
