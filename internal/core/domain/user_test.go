package domain

import "testing"

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{
			name:     "Valid Patient Role",
			role:     RolePatient,
			expected: true,
		},
		{
			name:     "Valid Doctor Role",
			role:     RoleDoctor,
			expected: true,
		},
		{
			name:     "Valid Admin Role",
			role:     RoleAdmin,
			expected: true,
		},
		{
			name:     "Invalid Role",
			role:     Role("invalid_role"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.expected {
				t.Errorf("Role.IsValid() for %s = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}
