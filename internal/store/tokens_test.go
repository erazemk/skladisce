package store

import (
	"context"
	"testing"
	"time"

	"github.com/erazemk/skladisce/internal/db"
)

func TestRevokeAndCheckToken(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	// Token should not be revoked initially.
	revoked, err := IsTokenRevoked(ctx, database, "test-jti-1")
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if revoked {
		t.Error("expected token not to be revoked")
	}

	// Revoke the token.
	err = RevokeToken(ctx, database, "test-jti-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// Now it should be revoked.
	revoked, err = IsTokenRevoked(ctx, database, "test-jti-1")
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if !revoked {
		t.Error("expected token to be revoked")
	}

	// Different JTI should not be revoked.
	revoked, err = IsTokenRevoked(ctx, database, "test-jti-2")
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if revoked {
		t.Error("expected different token not to be revoked")
	}
}

func TestRevokeTokenIdempotent(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	// Revoking the same token twice should not error (INSERT OR IGNORE).
	err := RevokeToken(ctx, database, "test-jti-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("first RevokeToken: %v", err)
	}

	err = RevokeToken(ctx, database, "test-jti-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("second RevokeToken: %v", err)
	}
}
