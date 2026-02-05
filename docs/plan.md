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

| Component        | Choice                         | Reason                               |
| ---------------- | ------------------------------ | ------------------------------------ |
| Language          | Go                             | Requirement                          |
| Database          | SQLite via `modernc.org/sqlite` | Pure Go, no CGO needed              |
| Router            | `net/http` (Go 1.22+ ServeMux) | Pattern matching built-in, zero deps |
| Auth              | JWT (`golang-jwt/jwt/v5`)      | Stateless, simple                    |
| Password hashing  | `golang.org/x/crypto/bcrypt`   | Standard, battle-tested              |
| Frontend          | Go templates + htmx            | Server-rendered, ~14 KB JS, no build step |
| Static embedding  | `go:embed`                     | Single binary, no external files     |
| Build             | `CGO_ENABLED=0 go build`       | Static binary                        |

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
│   └── main.go                  — entry point, config, startup, seed-admin subcommand
├── internal/
│   ├── api/                     — JSON API handlers (/api/*)
│   │   ├── router.go            — API route registration
│   │   ├── middleware.go         — auth middleware, logging, CORS
│   │   ├── auth.go              — login handler (JSON)
│   │   ├── users.go             — user management handlers
│   │   ├── owners.go            — owner CRUD handlers
│   │   ├── items.go             — item CRUD + image handlers
│   │   ├── transfers.go         — transfer handlers
│   │   ├── inventory.go         — inventory/stock handlers
│   │   └── response.go          — JSON response helpers
│   ├── web/                     — page handlers (/*), server-rendered HTML
│   │   ├── router.go            — page route registration
│   │   ├── middleware.go         — cookie auth, redirect to /login
│   │   ├── templates.go         — template loading, rendering helpers
│   │   ├── auth.go              — GET/POST /login, logout
│   │   ├── dashboard.go         — GET /
│   │   ├── items.go             — item pages + htmx fragment handlers
│   │   ├── owners.go            — owner pages + htmx fragment handlers
│   │   ├── transfers.go         — transfer pages + htmx fragment handlers
│   │   └── users.go             — user management pages (admin)
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
├── web/
│   ├── static/
│   │   ├── htmx.min.js          — vendored htmx (~14 KB gzipped)
│   │   └── style.css            — minimal custom styles
│   ├── templates/
│   │   ├── layout.html          — base layout: head, nav (role-aware), footer
│   │   ├── login.html
│   │   ├── dashboard.html
│   │   ├── items.html
│   │   ├── item_detail.html
│   │   ├── owners.html
│   │   ├── owner_detail.html
│   │   ├── transfers.html
│   │   ├── transfer_new.html
│   │   └── users.html
│   └── embed.go                 — go:embed directives for static/ and templates/
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
| Expired JWT (browser)          | Cookie auth middleware redirects to `/login`; htmx sees 401 → redirect |
| htmx vs full page              | Handlers check `HX-Request` header; return fragment or full page      |

## Auth Flow

### JSON API (`/api/*`)

1. **No open registration.** Only admins can create users via `POST /api/users`.
2. First admin is seeded via CLI: `skladisce seed-admin` (prompts for
   username + password, creates an admin user directly in the DB).
3. `POST /api/auth/login` → returns JWT as JSON `{"token": "…"}`.
4. All other API endpoints require `Authorization: Bearer <token>` header.

### Browser UI (`/*`)

1. `GET /login` renders a login form.
2. `POST /login` validates credentials, sets a `HttpOnly; Secure; SameSite=Strict`
   cookie containing the JWT, and redirects to `/`.
3. All subsequent page requests carry the cookie automatically.
4. `POST /logout` clears the cookie and redirects to `/login`.
5. Expired/missing cookie → redirect to `/login`.
6. htmx requests that receive a 401 trigger a client-side redirect to `/login`
   (via `HX-Trigger` response header or a small event listener).

### Permission Summary

| Action                                             | Role      |
| -------------------------------------------------- | --------- |
| Manage users (create, update, delete)              | admin     |
| Manage items (create, edit, delete, image, status) | manager+  |
| Manage owners (create, edit, delete)               | manager+  |
| Manage stock (add stock, adjust quantities)        | manager+  |
| Create transfers (borrow, return, handoff)         | all roles |
| View inventory, history, details                   | all roles |

## Frontend

### Approach: Go Templates + htmx + `go:embed`

The frontend is server-rendered using Go's `html/template` package. All
templates, static assets (htmx, CSS), and any images are embedded into the
binary via `go:embed`. No build toolchain, no npm, no bundler.

**htmx** (~14 KB gzipped, vendored) handles dynamic interactions via HTML
attributes — no custom JavaScript required. It supports all HTTP methods
(`PUT`, `DELETE`) directly from HTML elements.

### Routing: Two Layers

The server exposes two sets of routes:

1. **JSON API** (`/api/*`) — pure REST, returns JSON. For programmatic access,
   scripts, or a future mobile app. Auth via `Authorization: Bearer <token>`.

2. **Page routes** (`/*`) — render HTML via Go templates. For the browser UI.
   Auth via JWT stored in an `HttpOnly` cookie (set on login). Page handlers
   check the `HX-Request` header to decide what to return:
   - **Normal request** → full page (layout + content).
   - **htmx request** (`HX-Request: true`) → HTML fragment (just the changed
     part of the page).

Both layers share the same `store` package — no logic duplication.

### Auth in the Browser

- `GET /login` renders a login form.
- `POST /login` validates credentials, sets an `HttpOnly` cookie with the JWT,
  and redirects to the dashboard.
- The cookie is sent automatically on every subsequent request.
- Logout clears the cookie.
- The JSON API continues to use `Authorization: Bearer` for non-browser clients.

### Role-Based Rendering

Templates receive the current user's role from the handler. Role checks are
**server-side only** — the HTML for privileged actions (delete buttons, stock
forms, user management links) is simply never rendered for unauthorized roles.
There is nothing to bypass client-side.

```html
<!-- Example: only managers+ see the delete button -->
{{if roleAtLeast .Role "manager"}}
<button hx-delete="/owners/{{.Owner.ID}}" hx-target="closest tr" hx-swap="outerHTML">
    Delete
</button>
{{end}}
```

### Pages

| Page           | Route               | Roles     | Description                                 |
| -------------- | ----------------    | --------- | ------------------------------------------- |
| Login          | `GET /login`        | public    | Username + password form                    |
| Dashboard      | `GET /`             | all       | Inventory overview, recent transfers        |
| Items          | `GET /items`        | all       | List items; manager+ sees add/edit/delete   |
| Item detail    | `GET /items/:id`    | all       | Distribution, history; manager+ sees edit   |
| Owners         | `GET /owners`       | all       | List people/locations; manager+ sees CRUD   |
| Owner detail   | `GET /owners/:id`   | all       | Inventory held; manager+ sees edit          |
| Transfers      | `GET /transfers`    | all       | Transfer log with filters                   |
| New transfer   | `GET /transfers/new`| all       | Form: pick item, from, to, quantity         |
| Users          | `GET /users`        | admin     | User management                             |
