# Contributing to ChatAPI

## Before You Start

Read `README.md` and `CLAUDE.md` to understand what ChatAPI is and what it is not. Contributions that add complexity without a clear use case for the target audience (developers building AI-powered apps) will not be merged.

## What We Welcome

- Bug fixes
- PostgreSQL repository adapter (`internal/repository/postgres/`)
- Redis broker implementation (`internal/broker/redis.go`)
- TypeScript SDK improvements
- Documentation improvements
- Example applications

## What We Won't Merge

- Horizontal scaling built into core (use the broker interface)
- Multi-tenancy
- MCP server
- Oversight / HITL primitives
- Webhook delivery
- Features that add config options without clear necessity

## Development Setup

```bash
git clone https://github.com/getchatapi/chatapi.git
cd chatapi
go mod download
cp .env.example .env   # set JWT_SECRET
go run ./cmd/chatapi
```

Requirements: Go 1.22+, GCC (for SQLite CGO).

## Running Tests

```bash
go test ./...
```

All tests run against real SQLite (in-memory, shared-cache). No mocks. No external services needed.

## Code Conventions

See `CLAUDE.md` for the full conventions. Key points:

- No ORMs. Raw `database/sql`.
- All SQL belongs in `internal/repository/sqlite/` — services never touch `*sql.DB`.
- Services depend on repository interfaces, not concrete types.
- Black-box tests: `package foo_test`, use `testutil.NewTestDB(t)`.
- No panics in library code.
- `slog` for logging — structured key-value pairs.

## Pull Requests

- One concern per PR.
- Tests required for new behaviour.
- Run `go test ./...` and `go build ./...` before opening a PR.
- Describe *why*, not just *what*, in the PR description.

## Security Issues

Please report security vulnerabilities privately by opening a GitHub issue marked **Security** rather than a public PR.
