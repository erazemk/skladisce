package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erazemk/skladisce/internal/auth"
	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/model"
	"github.com/erazemk/skladisce/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const testJWTSecret = "test-secret"

func setupTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	database := db.NewTestDB(t)
	router := NewRouter(database, testJWTSecret)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	// Create admin user.
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	store.CreateUser(ctx, database, "admin", string(hash), model.RoleAdmin)

	// Get token.
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "password"})
	resp, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d", resp.StatusCode)
	}

	var loginResp map[string]string
	json.NewDecoder(resp.Body).Decode(&loginResp)
	token := loginResp["token"]
	if token == "" {
		t.Fatal("empty token from login")
	}

	return server, token
}

func authRequest(method, url, token string, body any) (*http.Request, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func TestLoginEndpoint(t *testing.T) {
	server, _ := setupTestServer(t)

	// Test invalid credentials.
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	resp, _ := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad password, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOwnersAPIFlow(t *testing.T) {
	server, token := setupTestServer(t)

	// Create owner.
	req, _ := authRequest("POST", server.URL+"/api/owners", token, map[string]string{
		"name": "Storage Room",
		"type": model.OwnerTypeLocation,
	})
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// List owners.
	req, _ = authRequest("GET", server.URL+"/api/owners", token, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var owners []model.Owner
	json.NewDecoder(resp.Body).Decode(&owners)
	resp.Body.Close()
	if len(owners) != 1 {
		t.Errorf("expected 1 owner, got %d", len(owners))
	}
}

func TestItemsAPIFlow(t *testing.T) {
	server, token := setupTestServer(t)

	// Create item.
	req, _ := authRequest("POST", server.URL+"/api/items", token, map[string]string{
		"name":        "Laptop",
		"description": "Dell XPS",
	})
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// List items.
	req, _ = authRequest("GET", server.URL+"/api/items", token, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestUnauthenticatedAccess(t *testing.T) {
	database := db.NewTestDB(t)
	router := NewRouter(database, testJWTSecret)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resp, _ := http.Get(server.URL + "/api/items")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated request, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRoleBasedAccess(t *testing.T) {
	database := db.NewTestDB(t)
	router := NewRouter(database, testJWTSecret)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	// Create a regular user.
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	store.CreateUser(ctx, database, "user1", string(hash), model.RoleUser)

	userToken, _ := auth.GenerateToken(testJWTSecret, 1, "user1", model.RoleUser)

	// Regular user should not be able to create items (manager+ required).
	req, _ := authRequest("POST", server.URL+"/api/items", userToken, map[string]string{
		"name": "Test",
	})
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for user creating item, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Regular user should not access /api/users.
	req, _ = authRequest("GET", server.URL+"/api/users", userToken, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for user accessing users, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
