package model

import (
	"fmt"
	"time"
)

// User represents an authentication user (separate from owners).
type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// Roles.
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleUser    = "user"
)

// roleLevels maps roles to their privilege level. Unknown roles have level 0.
var roleLevels = map[string]int{
	RoleAdmin:   3,
	RoleManager: 2,
	RoleUser:    1,
}

// RoleAtLeast checks if role meets or exceeds the minimum required role.
// Returns false for any unknown role (fail-closed).
func RoleAtLeast(role, minimum string) bool {
	roleLevel, roleOK := roleLevels[role]
	minLevel, minOK := roleLevels[minimum]
	if !roleOK || !minOK {
		return false
	}
	return roleLevel >= minLevel
}

// MinPasswordLength is the minimum allowed password length.
const MinPasswordLength = 8

// ValidatePassword checks that a password meets minimum requirements.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	// bcrypt silently truncates at 72 bytes.
	if len([]byte(password)) > 72 {
		return fmt.Errorf("password must not exceed 72 bytes")
	}
	return nil
}
