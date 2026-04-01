# ChatAPI — Implementation Plan

## What ChatAPI Is

**The real-time communication layer between your AI and your humans.**

Self-hosted, open-source, single binary. Whether the human is a customer chatting with an AI copilot, an operator approving an agent action, or a support agent taking over from a bot — ChatAPI is the infrastructure underneath it all.

---

## Architectural Decisions

### Drop Multi-Tenancy

Multi-tenancy was designed for a hosted SaaS model. For a self-hosted product the deployer is the only tenant. Removing it eliminates:
- API keys in client code (security risk)
- Tenant isolation complexity
- A concept that confuses the mental model

**Replacement:** JWT-based auth. Your app authenticates your users and issues short-lived tokens. ChatAPI validates the JWT. Your server-side secret never touches the client. This is how Pusher and Ably work.

```
Your backend  →  signs JWT for authenticated user
User          →  connects to ChatAPI with JWT
ChatAPI       →  validates JWT (configurable secret or JWKS URL)
```

### SQLite — Intentionally

Single-process SQLite is a deliberate trade for simplicity. It means:
- Zero infrastructure dependencies
- Single binary deployment
- Trivial backup (copy a file)
- No connection pooling daemon

Horizontal scaling is a non-goal. If you need it, you've outgrown this tool.

### MCP Server Built-In

ChatAPI ships with an MCP (Model Context Protocol) server. Any MCP-compatible agent — Claude, Cursor, or any agent built on the protocol — can use ChatAPI with zero integration code. The agent discovers tools automatically:

```
send_message(room_id, content)
get_messages(room_id, limit, after_seq)
create_room(name, members)
is_user_online(user_id)
request_approval(room_id, action, context)
await_response(room_id, message_id, timeout)
```

This is the distribution strategy. Every developer dropping in an MCP client is a potential user.

---

## The Three Pillars

### 1. Real-Time Rooms
Rooms with any mix of human and AI participants. WebSocket delivery, persistent history, delivery guarantees, presence tracking. **Already built.**

### 2. Bot Participants
AI agents as first-class room members. Two modes:

**Webhook-driven** — ChatAPI POSTs inbound messages to your agent endpoint. Your agent processes with full capability (tools, RAG, multi-step) and replies via REST. You own the brain.

**Built-in LLM** — register a bot with a model config (provider, model, API key, system prompt). ChatAPI handles the LLM call, injects room history as context, streams the reply. Zero code required.

### 3. Oversight Primitives
Structured message types that make AI agents safe to deploy in production:
- Approval requests — agent asks a human before acting
- Escalation — bot hands off to a human participant
- Acknowledgement — human confirms they've seen something
- Audit trail — every agent decision and human response is logged and replayable

---

## Roadmap

### Phase 0 — Foundation Cleanup ← current state
- [x] Rooms, messages, WebSocket delivery
- [x] Presence tracking
- [x] Webhook notifications
- [x] Delivery retry worker
- [x] Test coverage across all services and handlers
- [ ] Remove multi-tenancy, replace with JWT auth
- [ ] Remove API keys from all flows

### Phase 1 — Bot Participants
- [ ] `POST /bots` — register a bot (webhook URL or LLM config)
- [ ] `POST /rooms/{id}/members` — add bot as participant (bot_id as user_id)
- [ ] Trigger bots on inbound message (goroutine, non-blocking)
- [ ] Webhook-driven bots: POST to endpoint, await reply, post as message
- [ ] Built-in LLM bots: OpenAI-compatible provider (covers OpenAI, Ollama, Groq, local models)
- [ ] Built-in LLM bots: Anthropic provider
- [ ] Streaming WebSocket events: `message.stream.start`, `message.stream.delta`, `message.stream.end`
- [ ] Thread context injection: room history passed as LLM context automatically

### Phase 2 — MCP Server
- [ ] MCP server built into the binary (listens on separate port or same)
- [ ] Tools: `send_message`, `get_messages`, `create_room`, `is_user_online`
- [ ] Tools: `request_approval`, `await_response` (oversight primitives)
- [ ] MCP auth: same JWT validation as REST
- [ ] Documentation: how to connect Claude Desktop, Cursor, custom agents

### Phase 3 — Oversight Primitives
- [ ] Structured message types: approval request, approval response, escalation, ack
- [ ] Room state: pending / active / resolved
- [ ] `await_response` blocks until human replies or timeout
- [ ] Audit log endpoint: full trace of agent decisions and human responses
- [ ] Webhook on approval/rejection (for agent to resume workflow)

### Phase 4 — Developer Experience
- [ ] JS client widget: streaming cursor, typing indicator, approval UI
- [ ] Quickstart: zero to working AI chat in 10 minutes
- [ ] Example repo: Next.js app with ChatAPI + Claude
- [ ] Example: autonomous agent with human-in-the-loop approval via MCP

---

## Data Model Changes

### Remove
```sql
DROP TABLE tenants;
-- Remove tenant_id columns from all tables (or keep as single-value for forward compat)
```

### Add
```sql
CREATE TABLE bots (
    bot_id         TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    mode           TEXT NOT NULL,        -- 'webhook' | 'llm'
    -- webhook
    webhook_url    TEXT,
    webhook_secret TEXT,
    -- llm
    provider       TEXT,                 -- 'openai' | 'anthropic'
    base_url       TEXT,                 -- for ollama / openai-compatible endpoints
    model          TEXT,
    api_key        TEXT,
    system_prompt  TEXT,
    max_context    INT DEFAULT 20,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## What We Don't Build

- Horizontal scaling / Redis pub-sub / multi-instance WebSocket
- A hosted SaaS version (open source only)
- A visual chat builder or no-code tool
- An agent framework or LLM orchestration layer
- Multi-tenancy (single workspace per deployment)
- Competing with Slack, Discord, or any end-user chat product
