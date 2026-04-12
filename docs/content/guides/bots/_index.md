---
title: "AI Bots"
weight: 11
---

# AI Bots

A bot in ChatAPI is an AI participant registered with an LLM provider. When a user sends a message in a room the bot belongs to, ChatAPI calls the LLM, streams the response back via `message.stream.*` events, and stores the final message. No agent process required.

---

## Setup

### 1. Set the API key on the server

The key lives in an environment variable — never in the database:

```env
GEMINI_API_KEY=AIza...
```

ChatAPI supports any provider with an OpenAI-compatible `/chat/completions` endpoint:

| Provider | `llm_base_url` |
|---|---|
| Gemini | `https://generativelanguage.googleapis.com/v1beta/openai/` |
| OpenAI | `https://api.openai.com/v1` |
| Ollama (local) | `http://localhost:11434/v1` |
| OpenRouter | `https://openrouter.ai/api/v1` |

### 2. Register the bot

```bash
curl -X POST http://localhost:8080/bots \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Support Bot",
    "llm_base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
    "llm_api_key_env": "GEMINI_API_KEY",
    "model": "gemini-2.0-flash"
  }'
```

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Display name |
| `llm_base_url` | Yes | OpenAI-compatible base URL |
| `llm_api_key_env` | Yes | Name of the env var holding the API key |
| `model` | Yes | Model identifier (e.g. `gemini-2.0-flash`, `gpt-4o`) |

### 3. Add the bot to a room

```bash
curl -X POST http://localhost:8080/rooms/room_abc123/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "bot_abc123"}'
```

The bot now responds to every message sent in that room.

---

## System prompt webhook

`WEBHOOK_URL` is **required** when using bots. ChatAPI calls it before every LLM request — your app returns the system prompt for that call. This is where your RAG pipeline, persona instructions, and customer context live. Because it's in your app, you can change the system prompt at any time without touching ChatAPI.

```env
WEBHOOK_URL=https://yourapp.com/api/chatapi/webhook
WEBHOOK_SECRET=your-hmac-secret   # recommended — used to verify the request came from ChatAPI
```

> Without `WEBHOOK_URL`, bots will call the LLM with no system prompt — the model has no instructions and will behave unpredictably.

### Request

ChatAPI sends a `POST` to `WEBHOOK_URL` with `type: "bot.context"` when a bot is about to respond:

```json
{
  "type": "bot.context",
  "bot_id": "bot_abc123",
  "room_id": "room_abc123",
  "message": {
    "message_id": "msg_def456",
    "sender_id": "alice",
    "content": "What is the refund policy?",
    "created_at": "2026-04-11T10:00:00Z"
  },
  "history": [
    { "role": "user",      "content": "Hi there" },
    { "role": "assistant", "content": "Hello! How can I help?" },
    { "role": "user",      "content": "What is the refund policy?" }
  ]
}
```

`history` contains up to 20 of the most recent messages in the room, formatted as OpenAI role/content pairs. The last entry is always the message that triggered the bot.

The same `WEBHOOK_URL` also receives `type: "message.offline"` events for push-notification delivery. Distinguish them by the `type` field.

### Response

Your webhook must return:

```json
{
  "system_prompt": "You are a support agent for Acme Corp.\n\nRefund policy: ..."
}
```

ChatAPI uses `system_prompt` as the `system` message at the top of the LLM messages array.

To silence the bot entirely — no LLM call, no stream events — return `skip: true`:

```json
{
  "skip": true
}
```

Use this for human escalation: when a human agent takes over a conversation, return `skip: true` and the bot goes silent for that message. The bot remains in the room and will respond again if `skip` is not set on future messages.

### Example — Next.js API route

```typescript
// app/api/chatapi/webhook/route.ts
export async function POST(req: Request) {
  const body = await req.json();

  // Handle offline push notifications
  if (body.type === "message.offline") {
    await sendPushNotification(body.recipient_id, body.message);
    return Response.json({ ok: true });
  }

  // Handle bot context injection (type === "bot.context")
  const { bot_id, room_id, message, history } = body;

  // Silence the bot if a human agent has taken over
  const room = await db.rooms.get(room_id);
  if (room.metadata?.escalated) {
    return Response.json({ skip: true });
  }

  const docs = await vectorSearch(message.content);
  const customer = await db.customers.findByUserId(message.sender_id);

  return Response.json({
    system_prompt: `You are a support agent for Acme Corp.
Tone: professional, concise.
Customer: ${customer.name} (plan: ${customer.plan})

Relevant knowledge base:
${docs.map(d => d.content).join('\n\n')}`,
  });
}
```

Your RAG pipeline, prompt engineering, and personalisation all live here. ChatAPI is the transport layer — your app is the brain.

---

## Streaming events

When a bot responds, clients receive:

| Event | Description |
|---|---|
| `message.stream.start` | Bot has started responding |
| `message.stream.delta` | One token chunk (repeats until done) |
| `message.stream.end` | Stream complete — message persisted with `seq` |
| `message.stream.error` | LLM call failed — discard any partial content |

See [WebSocket API](/api/websocket/) for the full event schema.

---

## Manage bots

```bash
# List all bots
curl http://localhost:8080/bots -H "Authorization: Bearer $TOKEN"

# Get a specific bot
curl http://localhost:8080/bots/bot_abc123 -H "Authorization: Bearer $TOKEN"

# Delete a bot
curl -X DELETE http://localhost:8080/bots/bot_abc123 -H "Authorization: Bearer $TOKEN"
```

---

## Next steps

- [WebSocket API](/api/websocket/) — Full event reference including stream events
- [REST API](/api/rest/) — Bot and room endpoint reference
- [TypeScript SDK](/sdk/) — `chat.bots.create(...)` and streaming event handlers
