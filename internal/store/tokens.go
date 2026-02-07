package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RevokeToken adds a token's JTI to the revocation list.
func RevokeToken(ctx context.Context, db *sql.DB, jti string, expiresAt time.Time) error {
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO revoked_tokens (jti, expires_at) VALUES (?, ?)`,
		jti, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}

	// Opportunistically clean up expired revocations.
	_, _ = db.ExecContext(ctx,
		`DELETE FROM revoked_tokens WHERE expires_at < ?`, time.Now(),
	)

	return nil
}

// IsTokenRevoked checks if a token's JTI has been revoked.
func IsTokenRevoked(ctx context.Context, db *sql.DB, jti string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM revoked_tokens WHERE jti = ?`, jti,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking token revocation: %w", err)
	}
	return count > 0, nil
}
