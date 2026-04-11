<p align="center">
  <img src="docs/static/logo.svg" width="240" alt="ChatAPI" />
</p>

<p align="center">
  The messaging layer for apps where AI is a participant.
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-00ED64?style=flat-square&labelColor=001E2B" alt="License" /></a>
  <a href="https://github.com/hastenr/chatapi/releases"><img src="https://img.shields.io/github/v/release/hastenr/chatapi?style=flat-square&color=00ED64&labelColor=001E2B" alt="Release" /></a>
  <a href="https://github.com/hastenr/chatapi/actions"><img src="https://img.shields.io/github/actions/workflow/status/hastenr/chatapi/ci.yml?style=flat-square&labelColor=001E2B" alt="CI" /></a>
</p>

<p align="center">
  <a href="https://docs.chatapi.cloud/">Docs</a> ·
  <a href="https://docs.chatapi.cloud/getting-started/">Quick Start</a> ·
  <a href="https://docs.chatapi.cloud/api/rest/">API Reference</a> ·
  <a href="https://docs.chatapi.cloud/guides/bots/">AI Bots</a>
</p>

---

Most chat infrastructure was built before AI was a participant in the conversation. Bolting LLM support onto those systems means wrestling with per-MAU pricing, vendor lock-in, and data leaving your infrastructure.

ChatAPI is built for the other case: apps where one or more participants is an AI. Your agent — whether it calls OpenAI, runs RAG, or does multi-step reasoning — connects to ChatAPI like any other user. ChatAPI handles the rest: real-time delivery, message history, presence, streaming, and offline webhooks. Single binary. Your data, your server.

## How it works

```
  Your users
     ↕  WebSocket
   ChatAPI
     ↕  REST / WebSocket (bot JWT)
  Your AI agent
     ↕
  OpenAI · Anthropic · Ollama · RAG · anything
```

Your agent is a normal process. It connects to ChatAPI with a JWT, receives messages, calls whatever LLM or pipeline it needs, and posts the reply back. ChatAPI streams the response to every connected client in real time. No vendor lock-in. No framework constraints. Swap models without touching your infrastructure.

## Features

- **Real-time WebSocket messaging** — DM and group rooms with presence, typing indicators, and at-least-once delivery guarantees
- **LLM streaming** — token-by-token responses over WebSocket via `message.stream.*` events
- **AI bots as first-class participants** — bots join rooms like users; your agent controls all the logic
- **JWT auth** — your backend signs tokens, ChatAPI validates them. No API keys, no sessions, no vendor accounts
- **Webhook for offline delivery** — ChatAPI calls your endpoint when a message arrives for an offline user, so you can trigger push notifications
- **TypeScript SDK** — `npm install @hastenr/chatapi-sdk`
- **Single binary** — SQLite included, no external services required at runtime
- **Portable** — swap SQLite → PostgreSQL or local pub/sub → Redis by implementing one interface. Zero service changes.

## Quick start

```bash
docker run -d \
  -p 8080:8080 \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -e ALLOWED_ORIGINS="*" \
  -v chatapi-data:/data \
  hastenr/chatapi:latest
```

```bash
curl http://localhost:8080/health
# {"status":"ok","db_writable":true}
```

Or from source (requires gcc for the SQLite driver):

```bash
git clone https://github.com/hastenr/chatapi.git
cd chatapi
cp .env.example .env    # set JWT_SECRET
go run ./cmd/chatapi
```

## Connect your agent in 5 minutes

**1. Register a bot**

```bash
TOKEN="<jwt signed with your JWT_SECRET>"

curl -X POST http://localhost:8080/bots \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Support Bot"}'
# {"bot_id": "bot_abc123", ...}
```

**2. Add the bot to a room**

```bash
curl -X POST http://localhost:8080/rooms/room_123/members \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"user_id": "bot_abc123"}'
```

**3. Connect your agent**

```javascript
// Mint a JWT with sub = bot_abc123, connect like any user
const ws = new WebSocket(`wss://your-server/ws?token=${botJWT}`);

ws.onmessage = async (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type !== "message") return;

  // Call your LLM, run RAG, do whatever you need
  const reply = await callYourPipeline(msg.content);

  ws.send(JSON.stringify({
    type: "send_message",
    data: { room_id: msg.room_id, content: reply },
  }));
};
```

Your users see the reply in real time. Message history is stored. Offline users get a webhook.

## Deploy

| Platform | |
|---|---|
| Docker Compose | `cp .env.example .env && docker compose up -d` |
| Fly.io | `fly launch` |
| Railway | Import repo, add a volume at `/data` |
| Binary | [Releases](https://github.com/hastenr/chatapi/releases) |

## Configuration

Two variables are required, everything else has a sensible default:

```env
JWT_SECRET=your-secret-here          # openssl rand -base64 32
ALLOWED_ORIGINS=https://yourapp.com  # required for browser clients
```

See [`.env.example`](.env.example) for the full reference.

## Scaling

Runs on a single $6 VPS out of the box. When you outgrow it:

- **PostgreSQL** — implement the repository interfaces. Zero service changes.
- **Horizontal scaling** — implement `broker.Broker` backed by Redis pub/sub. Zero service changes.

## License

MIT — see [LICENSE](LICENSE).
