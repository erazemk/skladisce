package model

import "time"

// Transfer represents an item movement between owners.
type Transfer struct {
	ID             int64     `json:"id"`
	ItemID         int64     `json:"item_id"`
	FromOwnerID    int64     `json:"from_owner_id"`
	ToOwnerID      int64     `json:"to_owner_id"`
	Quantity       int       `json:"quantity"`
	Notes          string    `json:"notes,omitempty"`
	TransferredAt  time.Time `json:"transferred_at"`
	TransferredBy  *int64    `json:"transferred_by,omitempty"`

	// Joined fields (not always populated).
	ItemName      string `json:"item_name,omitempty"`
	FromOwnerName string `json:"from_owner_name,omitempty"`
	ToOwnerName   string `json:"to_owner_name,omitempty"`
}

// Inventory represents the current quantity of an item held by an owner.
type Inventory struct {
	ItemID    int64  `json:"item_id"`
	OwnerID   int64  `json:"owner_id"`
	Quantity  int    `json:"quantity"`

	// Joined fields (not always populated).
	ItemName  string `json:"item_name,omitempty"`
	OwnerName string `json:"owner_name,omitempty"`
	OwnerType string `json:"owner_type,omitempty"`
}
