package db

import (
	"database/sql"
	"testing"
)

// NewTestDB creates a fresh in-memory SQLite database with the schema applied.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	if err := EnsureSchema(db); err != nil {
		db.Close()
		t.Fatalf("creating test database schema: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}
