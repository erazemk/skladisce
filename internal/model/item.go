package model

import "time"

// Item represents an item type (quantity-based, not individual tracking).
type Item struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	ImageMime   string     `json:"image_mime,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Item statuses.
const (
	ItemStatusActive  = "active"
	ItemStatusDamaged = "damaged"
	ItemStatusLost = "lost"
)
