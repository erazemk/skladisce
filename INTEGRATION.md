# Integration Guide

This guide is for developers building integrations against the Skladišče API
(e.g., Discord bots, scripts, dashboards). For the full system spec, see
[SPEC.md](SPEC.md).

## API Reference

The complete API is documented in **[openapi.json](openapi.json)** (OpenAPI 3.1).
Use it as the authoritative reference for all endpoints, request/response
schemas, and authentication requirements.

## Quick Start

### 1. Get a token

There is no open registration. An admin must create your account. Once you have
credentials:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username": "your_user", "password": "your_pass"}'
```

Response:
```json
{"token": "eyJhbGciOi..."}
```

### 2. Use the token

Pass it as a Bearer token on every request:

```bash
curl http://localhost:8080/api/items \
  -H 'Authorization: Bearer eyJhbGciOi...'
```

Tokens expire after 24 hours. Request a new one when you get a `401`.

### 3. Common operations

**List all items:**
```
GET /api/items
```

**List all owners (people and locations):**
```
GET /api/owners
GET /api/owners?type=person
GET /api/owners?type=location
```

**See what an owner holds:**
```
GET /api/owners/{id}/inventory
```

**Create a transfer (borrow/return/handoff):**
```
POST /api/transfers
{
  "item_id": 1,
  "from_owner_id": 2,
  "to_owner_id": 3,
  "quantity": 1,
  "notes": "Borrowed for the weekend"
}
```

**View transfer history:**
```
GET /api/transfers
GET /api/transfers?item_id=1
GET /api/transfers?owner_id=3
```

**Full inventory overview:**
```
GET /api/inventory
```

## Roles

Your account's role determines what you can do:

| Role      | Can do                                              |
| --------- | --------------------------------------------------- |
| `user`    | View everything, create transfers                   |
| `manager` | Above + create/edit/delete items and owners, manage stock |
| `admin`   | Above + create/edit/delete user accounts            |

A Discord bot that only needs to move items around works fine with a `user`
account. If it also needs to create new items or owners, use `manager`.

## Key Concepts

- **Owner**: either a `person` or a `location`. Items are always held by owners.
- **Transfer**: moves a quantity of an item from one owner to another. This is
  the only way items move — there is no separate borrow/return concept.
- **Inventory**: the current state — who holds how many of what.
- **Item status**: `active`, `damaged`, `lost`, or `removed` — informational
  only, doesn't block transfers.

## Error Handling

All errors return JSON:
```json
{"error": "description of what went wrong"}
```

Common status codes:
- `400` — bad request (missing fields, insufficient quantity, transfer to self)
- `401` — not authenticated (missing/expired token)
- `403` — insufficient permissions (wrong role)
- `404` — resource not found
- `409` — conflict (e.g., duplicate username)
