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
