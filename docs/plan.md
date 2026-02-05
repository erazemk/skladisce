# Skladišče — Inventory Management System

Backend service for tracking physical inventory items and who borrows them from
storage rooms. Built in Go with SQLite, compiled as a static binary (no CGO).

## Concept: Unified Owner Model

Every item movement is a **transfer** between **owners**. An owner is either a
`person` or a `location`. An item's quantity is always distributed across owners.

- **Borrowing** = transfer from location → person
- **Returning** = transfer from person → location
- **Handoff** = transfer from person → person

There is no separate borrow/return logic — it's all transfers.

## Tech Stack

| Component       | Choice                         | Reason                               |
| --------------- | ------------------------------ | ------------------------------------ |
| Language         | Go                             | Requirement                          |
| Database         | SQLite via `modernc.org/sqlite` | Pure Go, no CGO needed              |
| Router           | `net/http` (Go 1.22+ ServeMux) | Pattern matching built-in, zero deps |
| Auth             | JWT (`golang-jwt/jwt/v5`)      | Stateless, simple                    |
| Password hashing | `golang.org/x/crypto/bcrypt`   | Standard, battle-tested              |
| Build            | `CGO_ENABLED=0 go build`       | Static binary                        |

## Database Schema

```sql
-- Authentication users (separate from owners — a person owner doesn't need a login)
-- No email — users are identified by username + password only.
-- Registration is disabled; only admins can create users via POST /api/users.
CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'manager', 'user')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME
);

-- Unified: people and storage locations
CREATE TABLE owners (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('person', 'location')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- Item types (quantity-based, not individual tracking)
CREATE TABLE items (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    image       BLOB,
    image_mime  TEXT,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'damaged', 'retired')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

-- Current distribution: who/where holds how many of what
CREATE TABLE inventory (
    item_id   INTEGER NOT NULL REFERENCES items(id),
    owner_id  INTEGER NOT NULL REFERENCES owners(id),
    quantity  INTEGER NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (item_id, owner_id)
);

-- Audit log: every movement
CREATE TABLE transfers (
    id             INTEGER PRIMARY KEY,
    item_id        INTEGER NOT NULL REFERENCES items(id),
    from_owner_id  INTEGER NOT NULL REFERENCES owners(id),
    to_owner_id    INTEGER NOT NULL REFERENCES owners(id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    notes          TEXT,
    transferred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    transferred_by INTEGER REFERENCES users(id)
);
```

### Key Design Decisions

- **`users` is separate from `owners`** — not every person in the system needs
  a login, and not every login maps to a person owner.
- **`inventory` is the current state** (denormalized for fast queries);
  **`transfers` is the audit log**.
- Both must stay in sync (wrapped in transactions).
- **Soft delete** via `deleted_at` on users, owners, and items — preserves all
  history.

## Roles & Permissions

| Role      | Permissions                                                                  |
| --------- | ---------------------------------------------------------------------------- |
| `admin`   | Everything + manage users (create, update, delete)                           |
| `manager` | Add/edit/delete items, manage stock & adjustments, manage owners + user perms |
| `user`    | View inventory, create transfers (borrow/return/handoff), view history       |

There is no open registration. Only admins can create new users. The first user
is created via a CLI seed command (see Auth Flow below).

## API Endpoints

### Auth

```
POST   /api/auth/login             — authenticate, get JWT token
```

### Users (admin only)

```
GET    /api/users                  — list users
POST   /api/users                  — create user (username + password + role)
GET    /api/users/:id              — get user
PUT    /api/users/:id              — update user (role, password reset)
DELETE /api/users/:id              — soft delete user
```

### Owners (manager+)

```
GET    /api/owners                 — list (filter by ?type=person|location)   [all roles]
POST   /api/owners                 — create person or location                [manager+]
GET    /api/owners/:id             — get owner details                        [all roles]
PUT    /api/owners/:id             — update owner                             [manager+]
DELETE /api/owners/:id             — soft delete (fails if holding inventory)  [manager+]
GET    /api/owners/:id/inventory   — what items this owner holds              [all roles]
```

### Items (manager+ for writes)

```
GET    /api/items                  — list (filter by ?status=active)          [all roles]
POST   /api/items                  — create item type                         [manager+]
GET    /api/items/:id              — get item details + distribution          [all roles]
PUT    /api/items/:id              — update item metadata/status              [manager+]
DELETE /api/items/:id              — soft delete                              [manager+]
PUT    /api/items/:id/image        — upload image (multipart)                 [manager+]
GET    /api/items/:id/image        — serve image blob                         [all roles]
GET    /api/items/:id/history      — transfer history for this item           [all roles]
```

### Transfers

```
POST   /api/transfers              — move N of item X from owner A → B        [all roles]
GET    /api/transfers              — list (filter by ?item_id, ?owner_id, …)  [all roles]
```

### Inventory

```
GET    /api/inventory              — full overview (all items × all holders)   [all roles]
POST   /api/inventory/stock        — add initial stock to a location           [manager+]
POST   /api/inventory/adjust       — adjust quantity (correct errors, losses)  [manager+]
```

## Project Structure

```
skladisce/
├── cmd/server/
│   └── main.go                  — entry point, config, startup
├── internal/
│   ├── api/
│   │   ├── router.go            — route registration
│   │   ├── middleware.go         — auth middleware, logging, CORS
│   │   ├── auth.go              — login handler
│   │   ├── users.go             — user management handlers
│   │   ├── owners.go            — owner CRUD handlers
│   │   ├── items.go             — item CRUD + image handlers
│   │   ├── transfers.go         — transfer handlers
│   │   ├── inventory.go         — inventory/stock handlers
│   │   └── response.go          — JSON response helpers
│   ├── db/
│   │   ├── db.go                — connection setup, pragmas
│   │   └── migrations.go        — schema migrations
│   ├── store/
│   │   ├── users.go             — user DB queries
│   │   ├── owners.go            — owner DB queries
│   │   ├── items.go             — item DB queries
│   │   ├── transfers.go         — transfer + inventory queries (transactional)
│   │   └── inventory.go         — inventory queries
│   ├── model/
│   │   ├── user.go
│   │   ├── owner.go
│   │   ├── item.go
│   │   └── transfer.go
│   └── auth/
│       └── jwt.go               — token generation/validation
├── docs/
│   └── plan.md                  — this document
├── Makefile
├── go.mod
└── README.md
```

## Edge Cases & Business Rules

| Edge case                      | Handling                                                              |
| ------------------------------ | --------------------------------------------------------------------- |
| Transfer more than held        | Reject: check `inventory.quantity >= requested` in transaction        |
| Transfer to self               | Reject: `from_owner_id != to_owner_id`                               |
| Delete owner holding items     | Reject: must transfer all items away first                            |
| Delete item with inventory     | Soft-delete only; inventory remains queryable for history             |
| Concurrent transfers           | SQLite serialized transactions; `BEGIN IMMEDIATE` to avoid SQLITE_BUSY |
| First user registration        | No open registration; seed admin via CLI: `skladisce seed-admin`      |
| Image upload                   | Validate MIME type (jpg/png/webp), enforce size limit (~5 MB)         |
| Quantity goes to 0             | Delete the `inventory` row (constraint: `quantity > 0`)               |
| Adjust for lost items          | Manager uses `/inventory/adjust` with negative delta + notes          |
| Status change to `retired`     | Informational flag; doesn't block transfers (admin decision)          |

## Auth Flow

1. **No open registration.** Only admins can create users via `POST /api/users`.
2. First admin is seeded via CLI: `skladisce seed-admin` (prompts for
   username + password, creates an admin user directly in the DB).
3. `POST /api/auth/login` → returns JWT with `{user_id, username, role, exp}`.
4. All other endpoints require `Authorization: Bearer <token>`.
5. **Admin-only:** user management (create, update, delete users).
6. **Manager+:** item CRUD, owner CRUD, stock management, inventory adjustments.
7. **All authenticated users:** view inventory, create transfers, view history.
