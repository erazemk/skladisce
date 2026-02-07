package store

import (
	"context"
	"testing"

	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
)

func TestCreateAndGetUser(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, err := CreateUser(ctx, database, "testuser", "hash123", model.RoleUser)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", user.Username)
	}
	if user.Role != model.RoleUser {
		t.Errorf("expected role 'user', got %q", user.Role)
	}

	got, err := GetUser(ctx, database, user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", got.Username)
	}
}

func TestGetUserByUsername(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	CreateUser(ctx, database, "alice", "hash", model.RoleAdmin)

	user, err := GetUserByUsername(ctx, database, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "alice" {
		t.Errorf("expected 'alice', got %q", user.Username)
	}

	missing, err := GetUserByUsername(ctx, database, "bob")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for missing user")
	}
}

func TestListUsers(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	CreateUser(ctx, database, "a", "hash", model.RoleUser)
	CreateUser(ctx, database, "b", "hash", model.RoleManager)

	users, err := ListUsers(ctx, database)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestDeleteUser(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, _ := CreateUser(ctx, database, "deleteme", "hash", model.RoleUser)
	DeleteUser(ctx, database, user.ID)

	users, _ := ListUsers(ctx, database)
	if len(users) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(users))
	}
}

func TestDeleteUserAndRecreateWithSameName(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, err := CreateUser(ctx, database, "reusable", "hash1", model.RoleUser)
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	if err := DeleteUser(ctx, database, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// Creating a new user with the same username should succeed.
	user2, err := CreateUser(ctx, database, "reusable", "hash2", model.RoleManager)
	if err != nil {
		t.Fatalf("second CreateUser with same username should succeed: %v", err)
	}
	if user2.Role != model.RoleManager {
		t.Errorf("expected role 'manager', got %q", user2.Role)
	}

	// GetUserByUsername should return the new active user, not the deleted one.
	got, err := GetUserByUsername(ctx, database, "reusable")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("expected active user, got nil")
	}
	if got.ID != user2.ID {
		t.Errorf("expected user ID %d, got %d", user2.ID, got.ID)
	}
}

func TestUpdateUserPassword(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, _ := CreateUser(ctx, database, "pwuser", "oldhash", model.RoleUser)
	UpdateUserPassword(ctx, database, user.ID, "newhash")

	got, _ := GetUser(ctx, database, user.ID)
	if got.PasswordHash != "newhash" {
		t.Errorf("expected password hash 'newhash', got %q", got.PasswordHash)
	}
}

func TestUpdateUserRole(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, _ := CreateUser(ctx, database, "roleuser", "hash", model.RoleUser)

	// Update role to manager.
	if err := UpdateUser(ctx, database, user.ID, model.RoleManager); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	got, _ := GetUser(ctx, database, user.ID)
	if got.Role != model.RoleManager {
		t.Errorf("expected role 'manager', got %q", got.Role)
	}

	// Update role to admin.
	if err := UpdateUser(ctx, database, user.ID, model.RoleAdmin); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	got, _ = GetUser(ctx, database, user.ID)
	if got.Role != model.RoleAdmin {
		t.Errorf("expected role 'admin', got %q", got.Role)
	}
}

func TestUpdateUserRoleNotFound(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	// Non-existent user should return error.
	if err := UpdateUser(ctx, database, 9999, model.RoleAdmin); err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestUpdateUserRoleDeletedUser(t *testing.T) {
	database := db.NewTestDB(t)
	ctx := context.Background()

	user, _ := CreateUser(ctx, database, "deleted", "hash", model.RoleUser)
	DeleteUser(ctx, database, user.ID)

	// Updating a soft-deleted user should return error.
	if err := UpdateUser(ctx, database, user.ID, model.RoleAdmin); err == nil {
		t.Error("expected error for deleted user, got nil")
	}
}
