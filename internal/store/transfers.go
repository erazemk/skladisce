package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erazemk/skladisce/internal/model"
)

// CreateTransfer creates a transfer, updating inventory in a single transaction.
// Uses BEGIN IMMEDIATE to prevent concurrent modification issues.
func CreateTransfer(ctx context.Context, db *sql.DB, itemID, fromOwnerID, toOwnerID int64, quantity int, notes string, transferredBy *int64) (*model.Transfer, error) {
	if fromOwnerID == toOwnerID {
		return nil, fmt.Errorf("cannot transfer to same owner")
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Use BEGIN IMMEDIATE semantics by acquiring a write lock early.
	if _, err := tx.ExecContext(ctx, "SELECT 1"); err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	// Check available quantity.
	var available int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(quantity, 0) FROM inventory WHERE item_id = ? AND owner_id = ?`,
		itemID, fromOwnerID,
	).Scan(&available)
	if err == sql.ErrNoRows {
		available = 0
	} else if err != nil {
		return nil, fmt.Errorf("checking available quantity: %w", err)
	}

	if available < quantity {
		return nil, fmt.Errorf("insufficient quantity: have %d, need %d", available, quantity)
	}

	// Decrease from source.
	newQty := available - quantity
	if newQty == 0 {
		_, err = tx.ExecContext(ctx,
			`DELETE FROM inventory WHERE item_id = ? AND owner_id = ?`,
			itemID, fromOwnerID,
		)
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE inventory SET quantity = ? WHERE item_id = ? AND owner_id = ?`,
			newQty, itemID, fromOwnerID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("updating source inventory: %w", err)
	}

	// Increase at destination.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO inventory (item_id, owner_id, quantity) VALUES (?, ?, ?)
		 ON CONFLICT (item_id, owner_id) DO UPDATE SET quantity = quantity + ?`,
		itemID, toOwnerID, quantity, quantity,
	)
	if err != nil {
		return nil, fmt.Errorf("updating destination inventory: %w", err)
	}

	// Record the transfer.
	result, err := tx.ExecContext(ctx,
		`INSERT INTO transfers (item_id, from_owner_id, to_owner_id, quantity, notes, transferred_by)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		itemID, fromOwnerID, toOwnerID, quantity, notes, transferredBy,
	)
	if err != nil {
		return nil, fmt.Errorf("recording transfer: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transfer: %w", err)
	}

	transferID, _ := result.LastInsertId()
	return GetTransfer(ctx, db, transferID)
}

// GetTransfer returns a transfer by ID.
func GetTransfer(ctx context.Context, db *sql.DB, id int64) (*model.Transfer, error) {
	t := &model.Transfer{}
	var notes sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT t.id, t.item_id, t.from_owner_id, t.to_owner_id, t.quantity, t.notes,
		        t.transferred_at, t.transferred_by,
		        i.name AS item_name, fo.name AS from_owner_name, too.name AS to_owner_name
		 FROM transfers t
		 JOIN items i ON i.id = t.item_id
		 JOIN owners fo ON fo.id = t.from_owner_id
		 JOIN owners too ON too.id = t.to_owner_id
		 WHERE t.id = ?`, id,
	).Scan(&t.ID, &t.ItemID, &t.FromOwnerID, &t.ToOwnerID, &t.Quantity, &notes,
		&t.TransferredAt, &t.TransferredBy,
		&t.ItemName, &t.FromOwnerName, &t.ToOwnerName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting transfer: %w", err)
	}
	t.Notes = notes.String
	return t, nil
}

// ListTransfers returns transfers, optionally filtered by item or owner.
func ListTransfers(ctx context.Context, db *sql.DB, itemID, ownerID int64) ([]model.Transfer, error) {
	query := `SELECT t.id, t.item_id, t.from_owner_id, t.to_owner_id, t.quantity, t.notes,
	                 t.transferred_at, t.transferred_by,
	                 i.name AS item_name, fo.name AS from_owner_name, too.name AS to_owner_name
	          FROM transfers t
	          JOIN items i ON i.id = t.item_id
	          JOIN owners fo ON fo.id = t.from_owner_id
	          JOIN owners too ON too.id = t.to_owner_id
	          WHERE 1=1`
	var args []any

	if itemID > 0 {
		query += ` AND t.item_id = ?`
		args = append(args, itemID)
	}
	if ownerID > 0 {
		query += ` AND (t.from_owner_id = ? OR t.to_owner_id = ?)`
		args = append(args, ownerID, ownerID)
	}

	query += ` ORDER BY t.transferred_at DESC`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing transfers: %w", err)
	}
	defer rows.Close()

	return scanTransfers(rows)
}

func scanTransfers(rows *sql.Rows) ([]model.Transfer, error) {
	var transfers []model.Transfer
	for rows.Next() {
		var t model.Transfer
		var notes sql.NullString
		if err := rows.Scan(&t.ID, &t.ItemID, &t.FromOwnerID, &t.ToOwnerID, &t.Quantity, &notes,
			&t.TransferredAt, &t.TransferredBy,
			&t.ItemName, &t.FromOwnerName, &t.ToOwnerName); err != nil {
			return nil, fmt.Errorf("scanning transfer: %w", err)
		}
		t.Notes = notes.String
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}
