package rbac

import (
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestRoleRank tests the internal role ranking function.
func TestRoleRank(t *testing.T) {
	tests := []struct {
		name     string
		role     models.Role
		expected int
	}{
		{"Admin should have rank 3", models.RoleAdmin, 3},
		{"User should have rank 2", models.RoleUser, 2},
		{"Guest should have rank 1", models.RoleGuest, 1},
		{"Unknown role should have rank 0", models.Role("unknown"), 0},
		{"Empty role should have rank 0", models.Role(""), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := roleRank(tt.role)
			assert.Equal(t, tt.expected, rank)
		})
	}
}

// TestHasRole tests role hierarchy checking.
func TestHasRole(t *testing.T) {
	t.Run("admin should have admin role", func(t *testing.T) {
		result := HasRole(models.RoleAdmin, models.RoleAdmin)
		assert.True(t, result)
	})

	t.Run("admin should have user role", func(t *testing.T) {
		result := HasRole(models.RoleAdmin, models.RoleUser)
		assert.True(t, result, "Admin should satisfy User role requirement")
	})

	t.Run("admin should have guest role", func(t *testing.T) {
		result := HasRole(models.RoleAdmin, models.RoleGuest)
		assert.True(t, result, "Admin should satisfy Guest role requirement")
	})

	t.Run("user should have user role", func(t *testing.T) {
		result := HasRole(models.RoleUser, models.RoleUser)
		assert.True(t, result)
	})

	t.Run("user should have guest role", func(t *testing.T) {
		result := HasRole(models.RoleUser, models.RoleGuest)
		assert.True(t, result, "User should satisfy Guest role requirement")
	})

	t.Run("user should NOT have admin role", func(t *testing.T) {
		result := HasRole(models.RoleUser, models.RoleAdmin)
		assert.False(t, result, "User should NOT satisfy Admin role requirement (privilege escalation)")
	})

	t.Run("guest should have guest role", func(t *testing.T) {
		result := HasRole(models.RoleGuest, models.RoleGuest)
		assert.True(t, result)
	})

	t.Run("guest should NOT have user role", func(t *testing.T) {
		result := HasRole(models.RoleGuest, models.RoleUser)
		assert.False(t, result, "Guest should NOT satisfy User role requirement (privilege escalation)")
	})

	t.Run("guest should NOT have admin role", func(t *testing.T) {
		result := HasRole(models.RoleGuest, models.RoleAdmin)
		assert.False(t, result, "Guest should NOT satisfy Admin role requirement (privilege escalation)")
	})
}

// TestHasRole_EdgeCases tests edge cases and security scenarios.
func TestHasRole_EdgeCases(t *testing.T) {
	t.Run("unknown role should not have any valid role", func(t *testing.T) {
		unknownRole := models.Role("hacker")

		assert.False(t, HasRole(unknownRole, models.RoleAdmin))
		assert.False(t, HasRole(unknownRole, models.RoleUser))
		assert.False(t, HasRole(unknownRole, models.RoleGuest))
	})

	t.Run("valid role with unknown required role returns true due to rank comparison", func(t *testing.T) {
		unknownRole := models.Role("superadmin")

		// NOTE: Current implementation returns true because unknown roles have rank 0
		// and Admin (rank 3) >= 0. This could be a security concern.
		assert.True(t, HasRole(models.RoleAdmin, unknownRole),
			"Current implementation: Admin (rank 3) >= unknown (rank 0)")
	})

	t.Run("empty string role should not have any privileges", func(t *testing.T) {
		emptyRole := models.Role("")

		assert.False(t, HasRole(emptyRole, models.RoleAdmin))
		assert.False(t, HasRole(emptyRole, models.RoleUser))
		assert.False(t, HasRole(emptyRole, models.RoleGuest))
	})

	t.Run("case sensitivity check", func(t *testing.T) {
		// Roles are case-sensitive, "Admin" != "admin"
		wrongCase := models.Role("Admin") // Should be "admin"

		assert.False(t, HasRole(wrongCase, models.RoleUser),
			"Case-sensitive role should not grant privileges")
	})
}

func TestHasRole_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		userRole     models.Role
		requiredRole models.Role
		expected     bool
	}{
		{"Admin->Admin", models.RoleAdmin, models.RoleAdmin, true},
		{"Admin->User", models.RoleAdmin, models.RoleUser, true},
		{"Admin->Guest", models.RoleAdmin, models.RoleGuest, true},

		{"User->Admin", models.RoleUser, models.RoleAdmin, false},
		{"User->User", models.RoleUser, models.RoleUser, true},
		{"User->Guest", models.RoleUser, models.RoleGuest, true},

		{"Guest->Admin", models.RoleGuest, models.RoleAdmin, false},
		{"Guest->User", models.RoleGuest, models.RoleUser, false},
		{"Guest->Guest", models.RoleGuest, models.RoleGuest, true},

		{"Unknown->Admin", models.Role("unknown"), models.RoleAdmin, false},
		{"Admin->Unknown", models.RoleAdmin, models.Role("unknown"), true}, // rank 3 >= rank 0
		{"Empty->Guest", models.Role(""), models.RoleGuest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasRole(tt.userRole, tt.requiredRole)
			assert.Equal(t, tt.expected, result)
		})
	}
}
