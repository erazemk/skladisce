# Agent Instructions

This project is developed by AI agents. Read this file first, then `SPEC.md`
for the full specification.

## Build & Verify Commands

```
make build          — CGO_ENABLED=0 go build -o skladisce ./cmd/server
make test           — go test -timeout 10s ./...
make lint           — go vet ./...
make run            — build + run with default flags
make clean          — remove binary
```

Always run `make build lint test` (single invocation) after changes.
Do not run them as separate commands.

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

- `make build lint test` — must all pass before committing
- New store functions need tests in the same package
- New API endpoints need integration tests with a test HTTP server

## Commit Conventions

- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `refactor:` code restructuring
- `test:` adding/updating tests

**Commit after every change.** Once `make build lint test` passes and the work
is complete, stage all changes and commit immediately. Do not wait for the user
to ask — committing is part of completing a task.

## Documentation as Source of Truth

Read `SPEC.md` before making architectural decisions. If spec and code disagree,
the spec wins. Update the spec first, then the code.

**Documentation-update rule:** After any significant code change, review and
update **all** affected documentation files in the same commit. Documentation
files include (but are not limited to):
- `README.md` — user-facing overview, install/usage instructions, examples
- `SPEC.md` — authoritative specification (behavior, schema, CLI, API, UI)
- `AGENTS.md` — agent instructions and conventions
- `openapi.json` — public API contract
- Any other `.md` files in the repository

Specifically, these changes require a documentation sweep:
- New or changed API endpoints, parameters, or responses
- New or changed business rules and edge cases
- New or changed CLI flags, arguments, or behavior
- New or changed frontend pages, routes, or UI behavior
- Changes to the project structure (new files/packages)
- Changes to build, test, or deployment procedures

**How to check:** Before committing, scan every documentation file listed above
for references to the changed functionality (CLI flags, endpoint paths, example
commands, etc.) and update any that are stale. Do not assume only `SPEC.md`
needs updating — `README.md` and other docs often duplicate information like
usage examples and must stay in sync.

**API spec rule:** Any change to API endpoints (new endpoints, changed
request/response schemas, changed parameters, changed auth requirements) must
also update `openapi.json`. This file is the public API contract used by
external integrators and their AI agents.

The spec is the single source of truth. Code without a matching spec entry is
undocumented behavior that may be removed.
