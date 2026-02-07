# Skladišče

Inventory management system for tracking physical items and who borrows them.
Built in Go with SQLite, compiled as a single static binary.

## Quick Start

```bash
make build
./skladisce
```

On first run, the database is created automatically and admin credentials are
printed to stdout. Save the password — it cannot be recovered.

## Usage

```bash
# Start with defaults (listens on :8080, DB at skladisce.sqlite3)
./skladisce

# Custom database path and listen address
./skladisce -db /data/skladisce.sqlite3 -a 127.0.0.1:8080

# All flags
./skladisce -h
```

### Flags

| Short | Long       | Default              | Description                        |
|-------|------------|----------------------|------------------------------------|
| `-d`  | `-db`      | `skladisce.sqlite3`  | SQLite database path               |
| `-a`  | `-addr`    | `:8080`              | Listen address (host:port)         |
| `-u`  | `-username`| `Admin`              | Admin username on first run        |
| `-h`  | `-help`    |                      | Show help and exit                 |

## Development

```bash
make build    # CGO_ENABLED=0 go build
make test     # go test -timeout 10s ./...
make lint     # go vet ./...
make run      # build + run
make clean    # remove binary
```

## Architecture

- **JSON API** (`/api/*`) — REST endpoints, Bearer token auth
- **Web UI** (`/*`) — Server-rendered HTML (Go templates + htmx), cookie auth
- **Database** — SQLite via `modernc.org/sqlite` (pure Go, no CGO)

See [SPEC.md](SPEC.md) for the full specification and [AGENTS.md](AGENTS.md)
for development conventions.
