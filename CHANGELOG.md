# Changelog

All notable changes to ChatAPI are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versions follow [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

### Breaking Changes
- **Auth**: API key authentication removed. All clients authenticate with JWT Bearer tokens (`Authorization: Bearer <token>`). Your backend signs JWTs with `JWT_SECRET`; ChatAPI validates them. The `sub` claim is the user ID.
- **Config**: `MASTER_API_KEY` removed. Set `JWT_SECRET` instead.
- **Config**: `DEFAULT_RATE_LIMIT` removed. Use `RATE_LIMIT_MESSAGES` and `RATE_LIMIT_MESSAGES_BURST`.
- **WebSocket**: Connect with `?token=<jwt>` instead of `?api_key=...`.
- **Room types**: `channel` type removed — use `group` instead. Only `dm` and `group` are supported.
- **Bots**: External bot mode removed. All bots are managed — ChatAPI calls the LLM directly on the bot's behalf. No separate agent process required.
- **Module path**: `github.com/hastenr/chatapi` → `github.com/getchatapi/chatapi`.

### Added
- **Managed AI bots**: Register a bot with an LLM provider URL via `POST /bots`. ChatAPI calls the model, streams tokens back via `message.stream.*` events, and stores the final reply. Works with any OpenAI-compatible endpoint (Gemini, OpenAI, Ollama, OpenRouter).
- **LLM streaming**: Bot replies stream token-by-token over WebSocket — `message.stream.start`, `message.stream.delta`, `message.stream.end`, `message.stream.error`.
- **Unified webhook**: Single `WEBHOOK_URL` handles two event types — `bot.context` (system prompt injection before each LLM call) and `message.offline` (push notification delivery for offline users). Distinguish by the `type` field.
- **Escalation support**: Webhook can return `{"skip": true}` to silence a bot when a human agent takes over. Bot resumes automatically when `skip` is no longer returned.
- **HMAC request signing**: ChatAPI signs all webhook requests with `WEBHOOK_SECRET` via `X-ChatAPI-Signature`. Verify on your end to ensure requests come from ChatAPI.
- **Message edit and delete**: `PUT /rooms/{room_id}/messages/{message_id}` and `DELETE /rooms/{room_id}/messages/{message_id}`. Only the original sender may edit or delete.
- **Room update**: `PATCH /rooms/{room_id}` — update room name or metadata.
- **Repository pattern**: All SQL lives in `internal/repository/sqlite/`. Services depend on interfaces — add PostgreSQL by implementing the same interfaces in a new adapter with zero service changes.
- **Broker interface**: `internal/broker/` decouples pub/sub from transport. Default `LocalBroker` is in-process. Implement `broker.Broker` backed by Redis to run multiple instances behind a load balancer.
- **Rate limiting**: Per-user message rate limiting with configurable sustained rate and burst (`RATE_LIMIT_MESSAGES`, `RATE_LIMIT_MESSAGES_BURST`).
- **Connection limits**: `WS_MAX_CONNECTIONS_PER_USER` caps concurrent WebSocket connections per user.
- **Dead-letter queue**: `GET /admin/dead-letters` lists messages that failed delivery after 5 retry attempts.
- **Test coverage**: Auth (JWT validation), broker, ratelimit, webhook (including HMAC signing), and all SQLite repositories (room, message, bot, delivery).
- **Issue templates**: GitHub bug report and feature request templates.
- **ROADMAP**: Public roadmap at `ROADMAP.md`.
- **Release pipeline**: Builds Linux (amd64/arm64), macOS (amd64/arm64), and Windows binaries on tag push. Docker image published to `hastenr/chatapi`.

### Removed
- MCP server — REST API is sufficient for agent integration.
- Oversight / HITL primitives (`request_approval`, `await_response`, room state machine) — deferred to a separate project.
- Multi-tenancy — single workspace per deployment.
- `IssueWSToken` / `ConsumeWSToken` — dead code from the old API key auth flow.
- `copilot-instructions.md` from `.github/`.

---

## [0.1.0] — 2025-12-20

### Added
- Initial release.
- Multi-tenant chat service with SQLite backend.
- REST API for rooms, messages, acknowledgments, and notifications.
- WebSocket API for real-time messaging, presence, and typing indicators.
- Durable message delivery with at-least-once guarantees and retry logic.
- Per-room monotonic message sequencing.
- Background delivery worker.
- Docker image and pre-built binaries for Linux, macOS, and Windows.
