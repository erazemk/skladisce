package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/erazemk/skladisce/internal/model"
)

// CreateItem creates a new item.
func CreateItem(ctx context.Context, db *sql.DB, name, description string) (*model.Item, error) {
	result, err := db.ExecContext(ctx,
		`INSERT INTO items (name, description) VALUES (?, ?)`,
		name, description,
	)
	if err != nil {
		return nil, fmt.Errorf("creating item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting item id: %w", err)
	}

	return GetItem(ctx, db, id)
}

// GetItem returns an item by ID.
func GetItem(ctx context.Context, db *sql.DB, id int64) (*model.Item, error) {
	item := &model.Item{}
	var description, imageMime sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT id, name, description, image_mime, status, created_at, updated_at, deleted_at
		 FROM items WHERE id = ?`, id,
	).Scan(&item.ID, &item.Name, &description, &imageMime, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}
	item.Description = description.String
	item.ImageMime = imageMime.String
	return item, nil
}

// ListItems returns all non-deleted items, optionally filtered by status.
func ListItems(ctx context.Context, db *sql.DB, status string) ([]model.Item, error) {
	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = db.QueryContext(ctx,
			`SELECT id, name, description, image_mime, status, created_at, updated_at, deleted_at
			 FROM items WHERE deleted_at IS NULL AND status = ? ORDER BY name`, status,
		)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id, name, description, image_mime, status, created_at, updated_at, deleted_at
			 FROM items WHERE deleted_at IS NULL ORDER BY name`,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listing items: %w", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		var description, imageMime sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &description, &imageMime, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt); err != nil {
			return nil, fmt.Errorf("scanning item: %w", err)
		}
		item.Description = description.String
		item.ImageMime = imageMime.String
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateItem updates an item's metadata.
func UpdateItem(ctx context.Context, db *sql.DB, id int64, name, description, status string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE items SET name = ?, description = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND deleted_at IS NULL`,
		name, description, status, id,
	)
	if err != nil {
		return fmt.Errorf("updating item: %w", err)
	}
	return nil
}

// DeleteItem soft-deletes an item.
func DeleteItem(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE items SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deleting item: %w", err)
	}
	return nil
}

// SetItemImage sets an item's image data.
func SetItemImage(ctx context.Context, db *sql.DB, id int64, image []byte, mime string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE items SET image = ?, image_mime = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND deleted_at IS NULL`,
		image, mime, id,
	)
	if err != nil {
		return fmt.Errorf("setting item image: %w", err)
	}
	return nil
}

// GetItemImage returns an item's image data and MIME type.
func GetItemImage(ctx context.Context, db *sql.DB, id int64) ([]byte, string, error) {
	var image []byte
	var mime sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT image, image_mime FROM items WHERE id = ?`, id,
	).Scan(&image, &mime)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("getting item image: %w", err)
	}
	return image, mime.String, nil
}

// GetItemHistory returns transfer history for an item.
func GetItemHistory(ctx context.Context, db *sql.DB, itemID int64) ([]model.Transfer, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT t.id, t.item_id, t.from_owner_id, t.to_owner_id, t.quantity, t.notes,
		        t.transferred_at, t.transferred_by,
		        i.name AS item_name, fo.name AS from_owner_name, too.name AS to_owner_name
		 FROM transfers t
		 JOIN items i ON i.id = t.item_id
		 JOIN owners fo ON fo.id = t.from_owner_id
		 JOIN owners too ON too.id = t.to_owner_id
		 WHERE t.item_id = ?
		 ORDER BY t.transferred_at DESC`, itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting item history: %w", err)
	}
	defer rows.Close()

	return scanTransfers(rows)
}
