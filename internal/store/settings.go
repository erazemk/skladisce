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
func GetJWTSecret(ctx context.Context, db *sql.DB) (string, error) {
	var secret string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = 'jwt_secret'").Scan(&secret)
	if err == nil {
		return secret, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("querying jwt_secret: %w", err)
	}

	// Generate a new secret.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating jwt secret: %w", err)
	}
	secret = hex.EncodeToString(buf)

	_, err = db.ExecContext(ctx, "INSERT INTO settings (key, value) VALUES ('jwt_secret', ?)", secret)
	if err != nil {
		return "", fmt.Errorf("storing jwt_secret: %w", err)
	}

	return secret, nil
}
