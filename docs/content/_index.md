+++
title = "ChatAPI Documentation"
type = "book"
weight = 1
+++

<p align="center">
  <img src="/logo.svg" width="220" alt="ChatAPI" />
</p>

<p align="center">Real-time chat infrastructure for AI-powered apps.</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go version" /></a>
  <a href="https://github.com/getchatapi/chatapi/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-00ED64?style=flat-square&labelColor=001E2B" alt="License" /></a>
  <a href="https://github.com/getchatapi/chatapi/releases"><img src="https://img.shields.io/github/v/release/getchatapi/chatapi?style=flat-square&color=00ED64&labelColor=001E2B" alt="Release" /></a>
  <a href="https://github.com/getchatapi/chatapi/actions"><img src="https://img.shields.io/github/actions/workflow/status/getchatapi/chatapi/ci.yml?style=flat-square&labelColor=001E2B" alt="CI" /></a>
</p>

<p align="center">
  <a href="/getting-started/">Quick Start</a> ·
  <a href="/api/rest/">API Reference</a> ·
  <a href="/guides/bots/">AI Bots</a> ·
  <a href="https://github.com/getchatapi/chatapi">GitHub</a>
</p>

---

ChatAPI is a self-hosted messaging server for apps where AI participates in conversations. It handles real-time delivery, message history, presence, typing indicators, and LLM streaming — so you focus on your product, not the plumbing.

## How it fits in your stack

### Messaging

```mermaid
flowchart LR
    A["User A"]
    CA["ChatAPI"]
    B["User B"]
    BE["Your Backend"]

    A -->|"send message"| CA
    CA -->|"deliver"| B
    CA -->|"message.offline"| BE
    BE -->|"email / push / SMS"| B
```

Users connect over WebSocket. ChatAPI stores messages, broadcasts to online room members, and fires a webhook when a recipient is offline — your backend decides how to notify them.

### AI Bots

```mermaid
flowchart LR
    C["Browser / Mobile"]
    CA["ChatAPI"]
    BE["Your Backend"]
    LLM["LLM Provider"]

    C <-->|"WebSocket"| CA
    CA -->|"bot.context"| BE
    BE -->|"system prompt"| CA
    CA -->|"chat/completions"| LLM
    LLM -->|"stream"| CA
    CA -->|"message.stream.*"| C
```

When a bot is in a room, ChatAPI calls your backend webhook to get the system prompt — your RAG pipeline, customer data, and escalation logic stay in your app. ChatAPI calls the LLM and streams tokens back to clients in real time.

## Built for

| Use case | How ChatAPI fits |
|---|---|
| **In-app messaging** | Add DM and group chat to any product — presence, typing indicators, and message history included |
| **Push notification relay** | Webhook fires when a message arrives for an offline user — forward to FCM, APNs, email, SMS, or any channel |
| **Customer support** | Bot handles tier-1 questions; human agent takes over with `skip: true` escalation |
| **In-app AI assistant** | Add an AI chat panel to any SaaS product — bot answers questions about your data via RAG in the webhook |
| **Sales & lead qualification** | Bot qualifies leads 24/7; human rep steps in when the lead is warm |
| **User onboarding** | Bot guides new users through setup; escalates to a human success manager for complex accounts |
| **Internal team AI** | Add AI bots to team rooms to answer questions about docs, policies, and internal data |

## Features

- **Managed AI bots** — register a bot with an LLM provider URL; ChatAPI calls the model, streams tokens back via `message.stream.*` events, and stores the reply
- **Real-time messaging** — rooms (DM or group) with presence, typing indicators, and at-least-once delivery
- **JWT auth** — your backend signs tokens, ChatAPI validates them; no vendor accounts, no sessions
- **Offline delivery** — messages queue for offline users; webhook fires so you can send push notifications
- **Escalation support** — webhook can return `{"skip": true}` to silence a bot when a human agent takes over
- **Single binary** — SQLite included, no external services required
- **Portable** — swap SQLite → PostgreSQL or local pub/sub → Redis by implementing one interface

## Documentation

- [Getting Started](/getting-started/) — Installation, configuration, and first API call
- [REST API](/api/rest/) — HTTP endpoint reference
- [WebSocket API](/api/websocket/) — Real-time event reference
- [AI Bots](/guides/bots/) — Register a bot and connect it to a room
- [Architecture](/architecture/) — System design and internals
