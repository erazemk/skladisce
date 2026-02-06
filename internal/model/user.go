package model

import "time"

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

// RoleAtLeast checks if role meets or exceeds the minimum required role.
func RoleAtLeast(role, minimum string) bool {
	levels := map[string]int{
		RoleAdmin:   3,
		RoleManager: 2,
		RoleUser:    1,
	}
	return levels[role] >= levels[minimum]
}
