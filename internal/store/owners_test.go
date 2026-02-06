package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
)

func TestCreateAndGetOwner(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	owner, err := CreateOwner(ctx, database, "Storage Room A", model.OwnerTypeLocation)
	if err != nil {
		t.Fatalf("CreateOwner: %v", err)
	}
	if owner.Name != "Storage Room A" {
		t.Errorf("expected name 'Storage Room A', got %q", owner.Name)
	}
	if owner.Type != model.OwnerTypeLocation {
		t.Errorf("expected type 'location', got %q", owner.Type)
	}

	got, _ := GetOwner(ctx, database, owner.ID)
	if got.Name != "Storage Room A" {
		t.Errorf("expected name 'Storage Room A', got %q", got.Name)
	}
}

func TestListOwnersFilterByType(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	CreateOwner(ctx, database, "Room", model.OwnerTypeLocation)
	CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)
	CreateOwner(ctx, database, "Closet", model.OwnerTypeLocation)

	all, _ := ListOwners(ctx, database, "")
	if len(all) != 3 {
		t.Errorf("expected 3 owners, got %d", len(all))
	}

	locations, _ := ListOwners(ctx, database, model.OwnerTypeLocation)
	if len(locations) != 2 {
		t.Errorf("expected 2 locations, got %d", len(locations))
	}

	people, _ := ListOwners(ctx, database, model.OwnerTypePerson)
	if len(people) != 1 {
		t.Errorf("expected 1 person, got %d", len(people))
	}
}

func TestDeleteOwnerWithInventoryFails(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	location, _ := CreateOwner(ctx, database, "Room", model.OwnerTypeLocation)
	item, _ := CreateItem(ctx, database, "Widget", "")
	AddStock(ctx, database, item.ID, location.ID, 5, nil)

	err := DeleteOwner(ctx, database, location.ID)
	if err == nil {
		t.Error("expected error deleting owner with inventory")
	}
}

func TestDeleteOwnerWithoutInventory(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	owner, _ := CreateOwner(ctx, database, "Empty Room", model.OwnerTypeLocation)
	err := DeleteOwner(ctx, database, owner.ID)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}
