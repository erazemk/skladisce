package model

import "testing"

func TestRoleAtLeast(t *testing.T) {
	tests := []struct {
		role     string
		minimum  string
		expected bool
	}{
		{RoleAdmin, RoleAdmin, true},
		{RoleAdmin, RoleManager, true},
		{RoleAdmin, RoleUser, true},
		{RoleManager, RoleAdmin, false},
		{RoleManager, RoleManager, true},
		{RoleManager, RoleUser, true},
		{RoleUser, RoleAdmin, false},
		{RoleUser, RoleManager, false},
		{RoleUser, RoleUser, true},
		// Unknown roles fail-closed.
		{"unknown", RoleUser, false},
		{RoleAdmin, "unknown", false},
		{"", "", false},
		{"", RoleUser, false},
	}

	for _, tt := range tests {
		got := RoleAtLeast(tt.role, tt.minimum)
		if got != tt.expected {
			t.Errorf("RoleAtLeast(%q, %q) = %v, want %v", tt.role, tt.minimum, got, tt.expected)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		wantErr  bool
	}{
		{"", true},
		{"short", true},
		{"1234567", true},
		{"12345678", false},
		{"a-valid-password", false},
	}

	for _, tt := range tests {
		err := ValidatePassword(tt.password)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
		}
	}
}
