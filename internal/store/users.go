package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erazemk/skladisce/internal/model"
)

// CreateUser creates a new user.
func CreateUser(ctx context.Context, db *sql.DB, username, passwordHash, role string) (*model.User, error) {
	result, err := db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role,
	)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting user id: %w", err)
	}

	return GetUser(ctx, db, id)
}

// GetUser returns a user by ID.
func GetUser(ctx context.Context, db *sql.DB, id int64) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, deleted_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return u, nil
}

// GetUserByUsername returns a user by username (including soft-deleted for auth checks).
func GetUserByUsername(ctx context.Context, db *sql.DB, username string) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at, deleted_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user by username: %w", err)
	}
	return u, nil
}

// ListUsers returns all non-deleted users.
func ListUsers(ctx context.Context, db *sql.DB) ([]model.User, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, username, password_hash, role, created_at, deleted_at
		 FROM users WHERE deleted_at IS NULL ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.DeletedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUser updates a user's role.
func UpdateUser(ctx context.Context, db *sql.DB, id int64, role string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET role = ? WHERE id = ? AND deleted_at IS NULL`,
		role, id,
	)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	return nil
}

// UpdateUserPassword updates a user's password hash.
func UpdateUserPassword(ctx context.Context, db *sql.DB, id int64, passwordHash string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ? AND deleted_at IS NULL`,
		passwordHash, id,
	)
	if err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// DeleteUser soft-deletes a user.
func DeleteUser(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}
