package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
)

func TestCreateAndGetItem(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, err := CreateItem(ctx, database, "Laptop", "Dell XPS 15")
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if item.Name != "Laptop" {
		t.Errorf("expected name 'Laptop', got %q", item.Name)
	}
	if item.Status != model.ItemStatusActive {
		t.Errorf("expected status 'active', got %q", item.Status)
	}
}

func TestListItemsByStatus(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	CreateItem(ctx, database, "Active Item", "")
	item2, _ := CreateItem(ctx, database, "Damaged Item", "")
	UpdateItem(ctx, database, item2.ID, "Damaged Item", "", model.ItemStatusDamaged)

	all, _ := ListItems(ctx, database, "")
	if len(all) != 2 {
		t.Errorf("expected 2 items, got %d", len(all))
	}

	active, _ := ListItems(ctx, database, model.ItemStatusActive)
	if len(active) != 1 {
		t.Errorf("expected 1 active item, got %d", len(active))
	}
}

func TestSoftDeleteItem(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Delete Me", "")
	DeleteItem(ctx, database, item.ID)

	items, _ := ListItems(ctx, database, "")
	if len(items) != 0 {
		t.Errorf("expected 0 items after soft delete, got %d", len(items))
	}

	// Should still be fetchable by ID (for history).
	got, _ := GetItem(ctx, database, item.ID)
	if got == nil {
		t.Error("expected soft-deleted item to still be fetchable by ID")
	}
}

func TestItemImage(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Photo Item", "")
	imageData := []byte("fake image data")
	SetItemImage(ctx, database, item.ID, imageData, "image/png")

	data, mime, err := GetItemImage(ctx, database, item.ID)
	if err != nil {
		t.Fatalf("GetItemImage: %v", err)
	}
	if string(data) != "fake image data" {
		t.Errorf("expected image data, got %q", string(data))
	}
	if mime != "image/png" {
		t.Errorf("expected mime 'image/png', got %q", mime)
	}
}
