<p align="center">
  <img src="docs/static/logo.svg" width="240" alt="ChatAPI" />
</p>

<p align="center">
  Real-time chat infrastructure for AI-powered apps.
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-00ED64?style=flat-square&labelColor=001E2B" alt="License" /></a>
  <a href="https://github.com/getchatapi/chatapi/releases"><img src="https://img.shields.io/github/v/release/getchatapi/chatapi?style=flat-square&color=00ED64&labelColor=001E2B" alt="Release" /></a>
  <a href="https://github.com/getchatapi/chatapi/actions"><img src="https://img.shields.io/github/actions/workflow/status/getchatapi/chatapi/ci.yml?style=flat-square&labelColor=001E2B" alt="CI" /></a>
</p>

<p align="center">
  <a href="https://docs.chatapi.cloud/">Docs</a> ·
  <a href="https://docs.chatapi.cloud/getting-started/">Quick Start</a> ·
  <a href="https://docs.chatapi.cloud/api/rest/">API Reference</a> ·
  <a href="https://docs.chatapi.cloud/guides/bots/">AI Bots</a>
</p>

---

ChatAPI is a self-hosted messaging server for apps where AI participates in conversations. It handles real-time delivery, message history, presence, typing indicators, and LLM streaming — so you focus on your product, not the plumbing.

Users connect over WebSocket. When a message is sent in a room that has a bot, ChatAPI calls your backend webhook to get the system prompt, then calls the LLM and streams the response back token by token. Your backend stays in control of context — RAG results, customer data, escalation logic — without managing a separate agent process.

## Features

- **Managed AI bots** — register a bot with an LLM provider URL; ChatAPI calls the model, streams tokens back via `message.stream.*` events, and stores the reply
- **Real-time messaging** — rooms (DM or group) with presence, typing indicators, and at-least-once delivery
- **JWT auth** — your backend signs tokens, ChatAPI validates them; no vendor accounts, no sessions
- **Offline delivery** — messages queue for offline users; webhook fires so you can send push notifications
- **Escalation support** — webhook can return `{"skip": true}` to silence a bot when a human agent takes over
- **Single binary** — SQLite included, no external services required
- **Portable** — swap SQLite → PostgreSQL or local pub/sub → Redis by implementing one interface

## Get started

Full setup guide, configuration reference, and API docs at **[docs.chatapi.cloud](https://docs.chatapi.cloud)**.

See [AI Bots](https://docs.chatapi.cloud/guides/bots/) in the docs for a full walkthrough.

## Deploy

Docker Compose, single binary, and reverse proxy setup — see [docs.chatapi.cloud/deploy](https://docs.chatapi.cloud/deploy/).

## Contributing

Requires Go 1.22+ and gcc (for the SQLite driver).

```bash
git clone https://github.com/getchatapi/chatapi.git
cd chatapi
go build -o bin/chatapi ./cmd/chatapi
export JWT_SECRET=$(openssl rand -base64 32)
export ALLOWED_ORIGINS="*"
./bin/chatapi
```

Run tests: `go test ./...`

## License

MIT — see [LICENSE](LICENSE).
