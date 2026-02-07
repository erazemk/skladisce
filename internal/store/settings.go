package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

// GetJWTSecret retrieves the JWT secret from the database.
// If no secret exists, it generates one, stores it, and returns it.
// Uses INSERT OR IGNORE + re-SELECT to avoid TOCTOU race on concurrent startup.
func GetJWTSecret(ctx context.Context, db *sql.DB) (string, error) {
	// Try to generate and insert first (safe against races).
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating jwt secret: %w", err)
	}
	candidate := hex.EncodeToString(buf)

	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('jwt_secret', ?)`,
		candidate,
	)
	if err != nil {
		return "", fmt.Errorf("storing jwt_secret: %w", err)
	}

	// Always read back (either our insert or the existing value).
	var secret string
	err = db.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = 'jwt_secret'`,
	).Scan(&secret)
	if err != nil {
		return "", fmt.Errorf("querying jwt_secret: %w", err)
	}

	return secret, nil
}
