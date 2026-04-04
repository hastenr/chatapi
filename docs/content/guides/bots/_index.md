---
title: "AI Bots"
weight: 11
---

# AI Bots

ChatAPI can run LLM-backed bots that automatically respond to messages in a room. When a new message arrives in a room that has a bot registered, ChatAPI fetches recent context, calls the LLM provider, and streams the response back to all room members in real time.

## Bot modes

| Mode | How it works |
|------|-------------|
| `llm` | ChatAPI calls the LLM on your behalf. You register the bot with an API key and model; ChatAPI handles context, streaming, and delivery. |
| `external` | The bot is a separate process that connects via WebSocket using its own JWT, just like a regular user. ChatAPI does not call any LLM — your process handles all logic. |

## Register an LLM bot

```bash
TOKEN="<your-signed-jwt>"

curl -X POST http://localhost:8080/bots \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Support Bot",
    "mode": "llm",
    "provider": "openai",
    "model": "gpt-4o",
    "api_key": "sk-...",
    "system_prompt": "You are a helpful support agent. Be concise.",
    "max_context": 20
  }'
```

Response:

```json
{
  "bot_id": "bot_abc123",
  "name": "Support Bot",
  "mode": "llm",
  "provider": "openai",
  "model": "gpt-4o",
  "system_prompt": "You are a helpful support agent. Be concise.",
  "max_context": 20,
  "created_at": "2026-04-02T12:00:00Z"
}
```

The `api_key` is stored but never returned after creation.

## Using OpenAI-compatible endpoints

Set `base_url` to point at any OpenAI-compatible API (Ollama, Groq, LM Studio, etc.):

```json
{
  "name": "Local Llama",
  "mode": "llm",
  "provider": "openai",
  "base_url": "http://localhost:11434/v1",
  "model": "llama3.2",
  "api_key": "ollama",
  "system_prompt": "You are a helpful assistant."
}
```

## Add a bot to a room

A bot is added to a room the same way any user is — as a member. Use the bot's ID as the `user_id`:

```bash
curl -X POST http://localhost:8080/rooms/room_abc123/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "bot_abc123"}'
```

Once added, the bot will respond to every message sent in the room.

## Streaming in action

When a user sends a message, connected WebSocket clients see the bot's response stream in real time:

1. **`message.stream.start`** — bot begins responding

```json
{
  "type": "message.stream.start",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789",
  "sender_id": "bot_abc123"
}
```

2. **`message.stream.delta`** — token chunks arrive

```json
{"type": "message.stream.delta", "room_id": "room_abc123", "message_id": "msg_stream_789", "delta": "Sure, "}
{"type": "message.stream.delta", "room_id": "room_abc123", "message_id": "msg_stream_789", "delta": "here's what I found..."}
```

3. **`message.stream.end`** — stream complete, message persisted

```json
{
  "type": "message.stream.end",
  "room_id": "room_abc123",
  "message_id": "msg_stream_789",
  "content": "Sure, here's what I found...",
  "seq": 5
}
```

Clients that miss the stream can fetch the completed message via `GET /rooms/{room_id}/messages?after_seq=<last_seen>`.

## External bots

For full control over bot logic, use `mode: "external"`. Your process connects to the WebSocket as a normal user (its `sub` claim is the bot's user ID) and handles messages however it likes:

```javascript
// Register bot with mode: "external" first, get bot_id
// Mint a JWT with sub = bot_id
const ws = new WebSocket(`wss://your-chatapi.com/ws?token=${botJWT}`);

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === "message" && msg.sender_id !== botId) {
    // process and reply
    ws.send(JSON.stringify({
      type: "send_message",
      data: { room_id: msg.room_id, content: "Got your message!" },
    }));
  }
};
```

## List and manage bots

```bash
# List all bots
curl http://localhost:8080/bots \
  -H "Authorization: Bearer $TOKEN"

# Get a specific bot
curl http://localhost:8080/bots/bot_abc123 \
  -H "Authorization: Bearer $TOKEN"

# Delete a bot
curl -X DELETE http://localhost:8080/bots/bot_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

## Next steps

- [WebSocket API](/api/websocket/) — Full event reference including `message.stream.*`
- [REST API](/api/rest/) — Bot and room endpoint reference
