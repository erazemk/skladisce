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
	}

	for _, tt := range tests {
		got := RoleAtLeast(tt.role, tt.minimum)
		if got != tt.expected {
			t.Errorf("RoleAtLeast(%q, %q) = %v, want %v", tt.role, tt.minimum, got, tt.expected)
		}
	}
}
