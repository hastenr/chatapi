+++
title = "API Playground"
weight = 23
draft = false
+++

# API Playground

Test the ChatAPI endpoints interactively using Swagger UI.

**Authentication**: Set your `Authorization: Bearer <jwt>` header in the Swagger UI "Authorize" dialog.

## Interactive API Documentation

<div id="swagger-ui"></div>

<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.7.2/swagger-ui.css" />
<script src="https://unpkg.com/swagger-ui-dist@5.7.2/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5.7.2/swagger-ui-standalone-preset.js"></script>

<script>
window.onload = function() {
  SwaggerUIBundle({
    url: '/api/openapi.yaml',
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
  });
};
</script>

## Common Test Scenarios

### Create a Room
```json
POST /rooms
{
  "type": "dm",
  "members": ["alice", "bob"]
}
```

### Send a Message
```json
POST /rooms/{room_id}/messages
{
  "content": "Hello from the API playground!"
}
```

### Register a Bot
```json
POST /bots
{
  "name": "Support Bot",
  "mode": "llm",
  "provider": "openai",
  "model": "gpt-4o",
  "api_key": "sk-...",
  "system_prompt": "You are a helpful support agent."
}
```

### Check Health
```
GET /health   (no auth required)
```

## WebSocket Testing

Browsers cannot set custom headers on WebSocket connections. Use the JWT query parameter:

```javascript
const jwt = "<your-signed-jwt>";
const ws = new WebSocket(`ws://localhost:8080/ws?token=${jwt}`);

ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'send_message',
    data: { room_id: 'your-room-id', content: 'Hello via WebSocket!' }
  }));
};
```

## Next Steps

- [REST API Reference](/api/rest/) — Complete endpoint documentation
- [WebSocket API Reference](/api/websocket/) — Real-time API documentation
- [Getting Started](/getting-started/) — Installation and setup guide
