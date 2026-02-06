package model

import "time"

// Owner represents a person or location that can hold inventory.
type Owner struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Owner types.
const (
	OwnerTypePerson   = "person"
	OwnerTypeLocation = "location"
)
