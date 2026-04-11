+++
title = "ChatAPI Documentation"
type = "book"
weight = 1
+++

<p align="center">
  <img src="/logo.svg" width="220" alt="ChatAPI" />
</p>

<p align="center">The messaging layer for apps where AI is a participant.</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go version" /></a>
  <a href="https://github.com/hastenr/chatapi/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-00ED64?style=flat-square&labelColor=001E2B" alt="License" /></a>
  <a href="https://github.com/hastenr/chatapi/releases"><img src="https://img.shields.io/github/v/release/hastenr/chatapi?style=flat-square&color=00ED64&labelColor=001E2B" alt="Release" /></a>
  <a href="https://github.com/hastenr/chatapi/actions"><img src="https://img.shields.io/github/actions/workflow/status/hastenr/chatapi/ci.yml?style=flat-square&labelColor=001E2B" alt="CI" /></a>
</p>

<p align="center">
  <a href="/getting-started/">Quick Start</a> ·
  <a href="/api/rest/">API Reference</a> ·
  <a href="/guides/bots/">AI Bots</a> ·
  <a href="https://github.com/hastenr/chatapi">GitHub</a>
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
- **Webhook for offline delivery** — ChatAPI calls your endpoint when a message arrives for an offline user
- **TypeScript SDK** — `npm install @hastenr/chatapi-sdk`
- **Single binary** — SQLite included, no external services required at runtime
- **Portable** — swap SQLite → PostgreSQL or local pub/sub → Redis by implementing one interface

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

## Documentation

- [Getting Started](/getting-started/) — Installation, configuration, and first API call
- [REST API](/api/rest/) — HTTP endpoint reference
- [WebSocket API](/api/websocket/) — Real-time event reference
- [TypeScript SDK](/sdk/) — `npm install @hastenr/chatapi-sdk`
- [Architecture](/architecture/) — System design and database schema
- [AI Bots](/guides/bots/) — Connect your agent
