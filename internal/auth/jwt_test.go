package auth

import (
	"testing"
	"time"

	"github.com/erazemk/skladisce/internal/model"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"

	token, err := GenerateToken(secret, 1, "admin", model.RoleAdmin)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ValidateToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("expected user_id 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", claims.Username)
	}
	if claims.Role != model.RoleAdmin {
		t.Errorf("expected role 'admin', got %q", claims.Role)
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	token, _ := GenerateToken("secret1", 1, "admin", model.RoleAdmin)

	_, err := ValidateToken("secret2", token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestValidateTokenInvalid(t *testing.T) {
	_, err := ValidateToken("secret", "not-a-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestTokenExpiry(t *testing.T) {
	// Just verify the expiry is set correctly.
	secret := "test"
	token, _ := GenerateToken(secret, 1, "test", "user")
	claims, _ := ValidateToken(secret, token)

	expiresAt := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(TokenExpiry)

	// Should be within a few seconds.
	diff := expectedExpiry.Sub(expiresAt)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("token expiry too far from expected: diff=%v", diff)
	}
}
