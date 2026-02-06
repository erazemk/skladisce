# Agent Instructions

This project is developed by AI agents. Read this file first, then `SPEC.md`
for the full specification.

## Build & Verify Commands

```
make build          — CGO_ENABLED=0 go build -o skladisce ./cmd/server
make test           — go test ./...
make lint           — go vet ./...
make run            — build + run serve with default flags
make clean          — remove binary
```

Always run `make build` after changes to ensure CGO_ENABLED=0 compilation.
Run `make test` and `make lint` before committing.

## Architecture

```
cmd/server/main.go → internal/api/  (JSON /api/*)  → internal/store/ → internal/db/
                   → internal/web/  (HTML /*)       → internal/store/ → internal/db/
```

Both API and web layers share the same `store` package — no logic duplication.

## Code Conventions

- All code in English; all template UI text in Slovenian.
- Error handling: return errors, don't panic. Use `fmt.Errorf("doing X: %w", err)`.
- HTTP handlers: parse input → call store → write response. No business logic
  in handlers.
- Store functions accept `context.Context` as first argument.
- Transactions: any operation touching `inventory` + `transfers` must be in a
  single `BEGIN IMMEDIATE` transaction.
- Models are plain structs with JSON tags. No ORM.
- Tests use a fresh in-memory SQLite database per test function.

## File Placement Rules

- New API endpoint → `internal/api/<resource>.go` + register in `router.go`
- New page → `internal/web/<resource>.go` + template in `web/templates/`
- New DB query → `internal/store/<resource>.go`
- New data type → `internal/model/<resource>.go`
- Schema change → `internal/db/migrations.go` (append new migration)

## Testing Requirements

- `make build` must compile with CGO_ENABLED=0
- `make test` — all tests must pass
- `make lint` — no warnings
- New store functions need tests in the same package
- New API endpoints need integration tests with a test HTTP server

## Commit Conventions

- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `refactor:` code restructuring
- `test:` adding/updating tests

## Documentation as Source of Truth

Read `SPEC.md` before making architectural decisions. If spec and code disagree,
the spec wins. Update the spec first, then the code.
