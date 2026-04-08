# CLAUDE.md

Instructions for Claude when working in this repository.

## Project Overview

ChatAPI is self-hosted, open source chat infrastructure for AI-powered apps. See `README.md` for positioning and `CONTRIBUTING.md` for what we will and won't merge.

## Project Structure

```
internal/
  broker/           # Broker interface + LocalBroker (swap for Redis)
  repository/
    repository.go   # All repository interfaces
    sqlite/         # SQLite implementations (room, message, delivery, notification, bot, tenant)
  models/           # Shared types only — no logic
  db/               # Schema, migrations, db.New()
  services/         # Business logic, one package per domain
    chatroom/       # Room + membership logic (depends on repository.RoomRepository)
    message/        # Message storage + sequencing (depends on repository.MessageRepository)
    delivery/       # Retry worker, offline queuing (depends on repository.DeliveryRepository)
    realtime/       # WebSocket registry + broadcast (depends on broker.Broker)
    bot/            # Bot registration + LLM triggering
    notification/   # Durable notifications
    tenant/         # Tenant management
    webhook/        # Outbound webhook calls
  handlers/
    rest/           # HTTP handlers
    ws/             # WebSocket handler
  transport/        # Server wiring, route registration
  testutil/         # Test helpers (NewTestDB)
  config/           # Config struct, env loading
  auth/             # JWT validation
```

## Coding Conventions

### General
- Standard library first. Add a dependency only when the stdlib alternative is materially worse.
- No ORMs. Raw `database/sql` throughout.
- No panics in library code. Return errors; let the handler decide the HTTP status.
- Interfaces exist at the repository layer (multiple DB adapters planned) and broker layer (Redis planned).

### Repository Pattern
- All SQL lives in `internal/repository/sqlite/`. Services never touch `*sql.DB` directly.
- Services depend on repository interfaces from `internal/repository/repository.go`.
- To add PostgreSQL: implement the same interfaces in `internal/repository/postgres/` — no service changes.
- SQLite uses `?` placeholders. PostgreSQL will use `$1, $2`.

### Packages and naming
- Constructor pattern: `NewService(deps...) *Service`
- Business logic lives in `internal/services/<name>/service.go`
- HTTP status decisions belong in handlers, not services
- Services return sentinel errors by string match (`"not found"`, `"forbidden"`) — keep it simple

### Error handling
```go
return fmt.Errorf("context: %w", err)   // wrap with context
```
HTTP error responses are flat JSON:
```json
{"error": "not_found", "message": "room not found"}
```

### Logging
- `log/slog` with structured key-value pairs
- `slog.Info`, `slog.Warn`, `slog.Error` — no `slog.Debug` in production paths

### HTTP
- Standard `net/http` ServeMux with `{path_param}` pattern syntax
- JSON in, JSON out
- Auth: JWT Bearer token. `sub` claim = userID.

### Concurrency
- `sync/atomic` for counters
- Goroutines for async work (delivery, bot triggers)
- Always pass `context.Context` for anything that should respect shutdown

### Database
- SQLite via `github.com/mattn/go-sqlite3`
- WAL mode enabled in `db.New()`
- No open read cursors while writing to the same table — collect rows into a slice first, close, then write
- Nullable TEXT columns: scan into `sql.NullString`, not `string`

## Testing

- Package: `package foo_test` (black-box testing)
- DB: always use `testutil.NewTestDB(t)` — named shared-cache in-memory SQLite, not `:memory:`
- No mock databases. Integration tests against real SQLite.
- Construct services via repositories: `chatroom.NewService(sqlite.NewRoomRepository(db.DB))`
- Run all tests: `go test ./...`
- Run one package: `go test -count=1 -timeout=30s ./internal/services/chatroom/`

## Keeping Docs in Sync

Any change that adds, removes, or modifies an API endpoint, request/response shape, auth scheme, or WebSocket event **must** update:

1. `docs/static/api/openapi.yaml` — the machine-readable spec (used for client generation)
2. The relevant page under `docs/content/api/` — `rest.md`, `websocket.md`, or `_index.md`

This applies to handler additions, route changes, model renames, and status code corrections.

## What We Don't Do

- No MCP server — REST API is sufficient for agent integration
- No oversight/HITL primitives — deferred to a separate project
- No hosted SaaS infrastructure in this repo
- No visual builders or no-code tooling
- No agent framework or LLM orchestration — ChatAPI is the communication layer, not the brain
- No multi-tenancy — single workspace per deployment
