package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
)

func TestAddStockAndListInventory(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	location, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, location.ID, 10, nil)

	inv, _ := ListInventory(ctx, database)
	if len(inv) != 1 {
		t.Fatalf("expected 1 inventory entry, got %d", len(inv))
	}
	if inv[0].Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", inv[0].Quantity)
	}
}

func TestAddStockToPersonFails(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	person, _ := CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)

	err := AddStock(ctx, database, item.ID, person.ID, 10, nil)
	if err == nil {
		t.Error("expected error adding stock to person")
	}
}

func TestAddStockUpserts(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	location, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, location.ID, 5, nil)
	AddStock(ctx, database, item.ID, location.ID, 3, nil)

	inv, _ := ListInventory(ctx, database)
	if len(inv) != 1 {
		t.Fatalf("expected 1 inventory entry, got %d", len(inv))
	}
	if inv[0].Quantity != 8 {
		t.Errorf("expected quantity 8, got %d", inv[0].Quantity)
	}
}

func TestAdjustInventory(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	location, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, location.ID, 10, nil)

	// Decrease by 3.
	err := AdjustInventory(ctx, database, item.ID, location.ID, -3, "lost items", nil)
	if err != nil {
		t.Fatalf("AdjustInventory: %v", err)
	}

	inv, _ := GetOwnerInventory(ctx, database, location.ID)
	if len(inv) != 1 || inv[0].Quantity != 7 {
		t.Errorf("expected quantity 7, got %v", inv)
	}
}

func TestAdjustInventoryToZeroRemovesRow(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	location, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, location.ID, 5, nil)

	err := AdjustInventory(ctx, database, item.ID, location.ID, -5, "all lost", nil)
	if err != nil {
		t.Fatalf("AdjustInventory: %v", err)
	}

	inv, _ := ListInventory(ctx, database)
	if len(inv) != 0 {
		t.Errorf("expected 0 inventory entries, got %d", len(inv))
	}
}

func TestAdjustInventoryNegativeResultFails(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	location, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, location.ID, 3, nil)

	err := AdjustInventory(ctx, database, item.ID, location.ID, -5, "too much", nil)
	if err == nil {
		t.Error("expected error for negative result")
	}
}

func TestGetItemDistribution(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	loc1, _ := CreateOwner(ctx, database, "Room A", model.OwnerTypeLocation)
	loc2, _ := CreateOwner(ctx, database, "Room B", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, loc1.ID, 5, nil)
	AddStock(ctx, database, item.ID, loc2.ID, 3, nil)

	dist, _ := GetItemDistribution(ctx, database, item.ID)
	if len(dist) != 2 {
		t.Fatalf("expected 2 distribution entries, got %d", len(dist))
	}

	total := 0
	for _, d := range dist {
		total += d.Quantity
	}
	if total != 8 {
		t.Errorf("expected total 8, got %d", total)
	}
}
