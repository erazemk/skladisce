package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erazemk/skladisce/internal/model"
)

// ListInventory returns the full inventory overview.
func ListInventory(ctx context.Context, db *sql.DB) ([]model.Inventory, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT inv.item_id, inv.owner_id, inv.quantity,
		        i.name AS item_name, o.name AS owner_name, o.type AS owner_type
		 FROM inventory inv
		 JOIN items i ON i.id = inv.item_id
		 JOIN owners o ON o.id = inv.owner_id
		 ORDER BY i.name, o.name`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing inventory: %w", err)
	}
	defer rows.Close()

	var items []model.Inventory
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(&inv.ItemID, &inv.OwnerID, &inv.Quantity, &inv.ItemName, &inv.OwnerName, &inv.OwnerType); err != nil {
			return nil, fmt.Errorf("scanning inventory: %w", err)
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}

// AddStock adds initial stock of an item to a location owner.
func AddStock(ctx context.Context, db *sql.DB, itemID, ownerID int64, quantity int, userID *int64) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Verify the owner is a location.
	var ownerType string
	err = tx.QueryRowContext(ctx,
		`SELECT type FROM owners WHERE id = ? AND deleted_at IS NULL`, ownerID,
	).Scan(&ownerType)
	if err == sql.ErrNoRows {
		return fmt.Errorf("owner not found")
	}
	if err != nil {
		return fmt.Errorf("checking owner: %w", err)
	}
	if ownerType != model.OwnerTypeLocation {
		return fmt.Errorf("stock can only be added to locations")
	}

	// Upsert inventory.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO inventory (item_id, owner_id, quantity) VALUES (?, ?, ?)
		 ON CONFLICT (item_id, owner_id) DO UPDATE SET quantity = quantity + ?`,
		itemID, ownerID, quantity, quantity,
	)
	if err != nil {
		return fmt.Errorf("adding stock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing stock addition: %w", err)
	}
	return nil
}

// AdjustInventory adjusts inventory quantity (for corrections/losses).
// Delta can be negative. If resulting quantity is 0, the row is deleted.
func AdjustInventory(ctx context.Context, db *sql.DB, itemID, ownerID int64, delta int, notes string, userID *int64) error {
	if delta == 0 {
		return fmt.Errorf("delta must be non-zero")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current quantity.
	var current int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(quantity, 0) FROM inventory WHERE item_id = ? AND owner_id = ?`,
		itemID, ownerID,
	).Scan(&current)
	if err == sql.ErrNoRows {
		current = 0
	} else if err != nil {
		return fmt.Errorf("checking current quantity: %w", err)
	}

	newQty := current + delta
	if newQty < 0 {
		return fmt.Errorf("adjustment would result in negative quantity: %d + %d = %d", current, delta, newQty)
	}

	if newQty == 0 {
		_, err = tx.ExecContext(ctx,
			`DELETE FROM inventory WHERE item_id = ? AND owner_id = ?`,
			itemID, ownerID,
		)
	} else if current == 0 {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO inventory (item_id, owner_id, quantity) VALUES (?, ?, ?)`,
			itemID, ownerID, newQty,
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE inventory SET quantity = ? WHERE item_id = ? AND owner_id = ?`,
			newQty, itemID, ownerID,
		)
	}
	if err != nil {
		return fmt.Errorf("adjusting inventory: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing adjustment: %w", err)
	}
	return nil
}

// GetItemDistribution returns inventory entries for a specific item.
func GetItemDistribution(ctx context.Context, db *sql.DB, itemID int64) ([]model.Inventory, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT inv.item_id, inv.owner_id, inv.quantity,
		        i.name AS item_name, o.name AS owner_name, o.type AS owner_type
		 FROM inventory inv
		 JOIN items i ON i.id = inv.item_id
		 JOIN owners o ON o.id = inv.owner_id
		 WHERE inv.item_id = ?
		 ORDER BY o.type, o.name`, itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting item distribution: %w", err)
	}
	defer rows.Close()

	var items []model.Inventory
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(&inv.ItemID, &inv.OwnerID, &inv.Quantity, &inv.ItemName, &inv.OwnerName, &inv.OwnerType); err != nil {
			return nil, fmt.Errorf("scanning inventory: %w", err)
		}
		items = append(items, inv)
	}
	return items, rows.Err()
}
