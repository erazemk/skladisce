package db

import (
	"database/sql"
	"fmt"
)

// schema is the full database schema.
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'manager', 'user')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
    ON users(username) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS owners (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('person', 'location')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS items (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    image       BLOB,
    image_mime  TEXT,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'damaged', 'lost', 'removed')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

CREATE TABLE IF NOT EXISTS inventory (
    item_id   INTEGER NOT NULL REFERENCES items(id),
    owner_id  INTEGER NOT NULL REFERENCES owners(id),
    quantity  INTEGER NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (item_id, owner_id)
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transfers (
    id             INTEGER PRIMARY KEY,
    item_id        INTEGER NOT NULL REFERENCES items(id),
    from_owner_id  INTEGER NOT NULL REFERENCES owners(id),
    to_owner_id    INTEGER NOT NULL REFERENCES owners(id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    notes          TEXT,
    transferred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    transferred_by INTEGER REFERENCES users(id)
);
`

// migrations is a list of SQL statements applied in order after schema creation.
// Each migration must be idempotent. Append new migrations at the end.
var migrations = []string{
	// Migration 1: Replace hard UNIQUE on username with a partial unique index
	// that only covers active (non-deleted) users so that soft-deleted usernames
	// can be reused.
	`DROP INDEX IF EXISTS sqlite_autoindex_users_1`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
	     ON users(username) WHERE deleted_at IS NULL`,
}

// Migrate runs the database schema migrations.
func Migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("running migration %d: %w", i+1, err)
		}
	}

	return nil
}
