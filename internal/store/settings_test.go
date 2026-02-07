package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
)

func TestGetJWTSecret_GeneratesAndPersists(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// First call should generate a secret.
	secret1, err := GetJWTSecret(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if secret1 == "" {
		t.Fatal("expected non-empty secret")
	}
	if len(secret1) != 64 { // 32 bytes = 64 hex chars
		t.Fatalf("expected 64 hex chars, got %d", len(secret1))
	}

	// Second call should return the same secret.
	secret2, err := GetJWTSecret(ctx, database)
	if err != nil {
		t.Fatal(err)
	}
	if secret1 != secret2 {
		t.Fatalf("expected same secret, got %q and %q", secret1, secret2)
	}
}
