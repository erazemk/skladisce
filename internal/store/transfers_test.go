package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
)

func TestTransferBasic(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	from, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)
	to, _ := CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)

	// Add stock first.
	AddStock(ctx, database, item.ID, from.ID, 10, nil)

	// Transfer 3 from Storage to Alice.
	transfer, err := CreateTransfer(ctx, database, item.ID, from.ID, to.ID, 3, "test transfer", nil)
	if err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}
	if transfer.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", transfer.Quantity)
	}

	// Check inventory.
	fromInv, _ := GetOwnerInventory(ctx, database, from.ID)
	if len(fromInv) != 1 || fromInv[0].Quantity != 7 {
		t.Errorf("expected Storage to have 7, got %v", fromInv)
	}

	toInv, _ := GetOwnerInventory(ctx, database, to.ID)
	if len(toInv) != 1 || toInv[0].Quantity != 3 {
		t.Errorf("expected Alice to have 3, got %v", toInv)
	}
}

func TestTransferInsufficientQuantity(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	from, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)
	to, _ := CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)

	AddStock(ctx, database, item.ID, from.ID, 5, nil)

	_, err := CreateTransfer(ctx, database, item.ID, from.ID, to.ID, 10, "", nil)
	if err == nil {
		t.Error("expected error for insufficient quantity")
	}
}

func TestTransferToSelfRejected(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	owner, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)

	AddStock(ctx, database, item.ID, owner.ID, 5, nil)

	_, err := CreateTransfer(ctx, database, item.ID, owner.ID, owner.ID, 1, "", nil)
	if err == nil {
		t.Error("expected error for transfer to self")
	}
}

func TestTransferRemovesZeroInventory(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item, _ := CreateItem(ctx, database, "Widget", "")
	from, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)
	to, _ := CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)

	AddStock(ctx, database, item.ID, from.ID, 5, nil)

	// Transfer all 5.
	_, err := CreateTransfer(ctx, database, item.ID, from.ID, to.ID, 5, "", nil)
	if err != nil {
		t.Fatalf("CreateTransfer: %v", err)
	}

	// Storage should have no inventory row.
	fromInv, _ := GetOwnerInventory(ctx, database, from.ID)
	if len(fromInv) != 0 {
		t.Errorf("expected empty inventory for storage, got %d entries", len(fromInv))
	}
}

func TestListTransfersFiltered(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	item1, _ := CreateItem(ctx, database, "Widget", "")
	item2, _ := CreateItem(ctx, database, "Gadget", "")
	from, _ := CreateOwner(ctx, database, "Storage", model.OwnerTypeLocation)
	to, _ := CreateOwner(ctx, database, "Alice", model.OwnerTypePerson)

	AddStock(ctx, database, item1.ID, from.ID, 10, nil)
	AddStock(ctx, database, item2.ID, from.ID, 10, nil)

	CreateTransfer(ctx, database, item1.ID, from.ID, to.ID, 2, "", nil)
	CreateTransfer(ctx, database, item2.ID, from.ID, to.ID, 3, "", nil)

	all, _ := ListTransfers(ctx, database, 0, 0)
	if len(all) != 2 {
		t.Errorf("expected 2 transfers, got %d", len(all))
	}

	byItem, _ := ListTransfers(ctx, database, item1.ID, 0)
	if len(byItem) != 1 {
		t.Errorf("expected 1 transfer for item1, got %d", len(byItem))
	}

	byOwner, _ := ListTransfers(ctx, database, 0, to.ID)
	if len(byOwner) != 2 {
		t.Errorf("expected 2 transfers for Alice, got %d", len(byOwner))
	}
}
