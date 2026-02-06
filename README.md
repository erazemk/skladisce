# Skladišče

Inventory management system for tracking physical items and who borrows them.
Built in Go with SQLite, compiled as a single static binary.

## Quick Start

```bash
make build
./skladisce serve
```

On first run, the database is created automatically and admin credentials are
printed to stdout. Save the password — it cannot be recovered.

## Commands

```bash
# Initialize database only (fails if DB already exists)
./skladisce init --db skladisce.sqlite3

# Start the server (auto-initializes DB if missing)
./skladisce serve --db skladisce.sqlite3 --addr :8080 --jwt-secret <secret>
```

## Development

```bash
make build    # CGO_ENABLED=0 go build
make test     # go test -timeout 10s ./...
make lint     # go vet ./...
make run      # build + run serve
make clean    # remove binary
```

## Architecture

- **JSON API** (`/api/*`) — REST endpoints, Bearer token auth
- **Web UI** (`/*`) — Server-rendered HTML (Go templates + htmx), cookie auth
- **Database** — SQLite via `modernc.org/sqlite` (pure Go, no CGO)

See [SPEC.md](SPEC.md) for the full specification and [AGENTS.md](AGENTS.md)
for development conventions.
