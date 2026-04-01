# CLAUDE.md

Instructions for Claude when working in this repository.

## Before Making Architectural Decisions

Read `VISION.md` for product positioning and `PLAN.md` for the current implementation plan.
The two documents together define what we are building and why.

## Project Structure

```
internal/
  models/         # Shared types only — no logic
  db/             # Schema, migrations, db.New()
  services/       # Business logic, one package per domain
    chatroom/
    message/
    delivery/
    realtime/
    tenant/       # Being removed — see PLAN.md
    notification/
    webhook/
  handlers/
    rest/         # HTTP handlers
    ws/           # WebSocket handler
  transport/      # Server wiring, route registration
  testutil/       # Test helpers (NewTestDB)
  config/         # Config struct, env loading
```

## Coding Conventions

### General
- Standard library first. Add a dependency only when the stdlib alternative is materially worse.
- No ORMs. Raw `database/sql` throughout.
- No panics in library code. Return errors; let the handler decide the HTTP status.
- Interfaces only when there are multiple implementations or tests require it. None currently exist.

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
- Auth: JWT validation (see PLAN.md — API keys are being removed)

### Concurrency
- `sync/atomic` for counters
- Goroutines for async work (delivery, bot triggers, webhook calls)
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
- Run all tests: `go test ./...`
- Run one package: `go test -count=1 -timeout=30s ./internal/services/chatroom/`

## What We Don't Do

- No horizontal scaling / Redis pub-sub / multi-instance coordination
- No hosted SaaS infrastructure in this repo
- No visual builders or no-code tooling
- No agent framework or LLM orchestration — ChatAPI is the communication layer, not the brain
