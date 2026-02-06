package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erazemk/skladisce/internal/model"
)

// CreateOwner creates a new owner (person or location).
func CreateOwner(ctx context.Context, db *sql.DB, name, ownerType string) (*model.Owner, error) {
	result, err := db.ExecContext(ctx,
		`INSERT INTO owners (name, type) VALUES (?, ?)`,
		name, ownerType,
	)
	if err != nil {
		return nil, fmt.Errorf("creating owner: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting owner id: %w", err)
	}

	return GetOwner(ctx, db, id)
}

// GetOwner returns an owner by ID.
func GetOwner(ctx context.Context, db *sql.DB, id int64) (*model.Owner, error) {
	o := &model.Owner{}
	err := db.QueryRowContext(ctx,
		`SELECT id, name, type, created_at, deleted_at
		 FROM owners WHERE id = ?`, id,
	).Scan(&o.ID, &o.Name, &o.Type, &o.CreatedAt, &o.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting owner: %w", err)
	}
	return o, nil
}

// ListOwners returns all non-deleted owners, optionally filtered by type.
func ListOwners(ctx context.Context, db *sql.DB, ownerType string) ([]model.Owner, error) {
	var rows *sql.Rows
	var err error

	if ownerType != "" {
		rows, err = db.QueryContext(ctx,
			`SELECT id, name, type, created_at, deleted_at
			 FROM owners WHERE deleted_at IS NULL AND type = ? ORDER BY name`, ownerType,
		)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id, name, type, created_at, deleted_at
			 FROM owners WHERE deleted_at IS NULL ORDER BY name`,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listing owners: %w", err)
	}
	defer rows.Close()

	var owners []model.Owner
	for rows.Next() {
		var o model.Owner
		if err := rows.Scan(&o.ID, &o.Name, &o.Type, &o.CreatedAt, &o.DeletedAt); err != nil {
			return nil, fmt.Errorf("scanning owner: %w", err)
		}
		owners = append(owners, o)
	}
	return owners, rows.Err()
}

// UpdateOwner updates an owner's name.
func UpdateOwner(ctx context.Context, db *sql.DB, id int64, name string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE owners SET name = ? WHERE id = ? AND deleted_at IS NULL`,
		name, id,
	)
	if err != nil {
		return fmt.Errorf("updating owner: %w", err)
	}
	return nil
}

// DeleteOwner soft-deletes an owner. Fails if the owner holds any inventory.
func DeleteOwner(ctx context.Context, db *sql.DB, id int64) error {
	// Check if owner holds inventory.
	var count int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory WHERE owner_id = ?`, id,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("checking owner inventory: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete owner: still holds %d inventory entries", count)
	}

	_, err = db.ExecContext(ctx,
		`UPDATE owners SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting owner: %w", err)
	}
	return nil
}

// GetOwnerInventory returns all inventory entries for an owner.
func GetOwnerInventory(ctx context.Context, db *sql.DB, ownerID int64) ([]model.Inventory, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT inv.item_id, inv.owner_id, inv.quantity, i.name AS item_name
		 FROM inventory inv
		 JOIN items i ON i.id = inv.item_id
		 WHERE inv.owner_id = ?
		 ORDER BY i.name`, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting owner inventory: %w", err)
	}
	defer rows.Close()

	var items []model.Inventory
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(&inv.ItemID, &inv.OwnerID, &inv.Quantity, &inv.ItemName); err != nil {
			return nil, fmt.Errorf("scanning inventory: %w", err)
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}
