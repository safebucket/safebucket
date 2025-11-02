package rbac

import (
	"testing"

	"api/internal/models"

	"github.com/stretchr/testify/assert"
)

// TestGroupRank tests the internal group ranking function.
func TestGroupRank(t *testing.T) {
	tests := []struct {
		name     string
		group    models.Group
		expected int
	}{
		{"Owner should have rank 3", models.GroupOwner, 3},
		{"Contributor should have rank 2", models.GroupContributor, 2},
		{"Viewer should have rank 1", models.GroupViewer, 1},
		{"Unknown group should have rank 0", models.Group("unknown"), 0},
		{"Empty group should have rank 0", models.Group(""), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rank := groupRank(tt.group)
			assert.Equal(t, tt.expected, rank)
		})
	}
}

// TestHasGroup tests group hierarchy checking.
func TestHasGroup(t *testing.T) {
	t.Run("owner should have owner group", func(t *testing.T) {
		result := HasGroup(models.GroupOwner, models.GroupOwner)
		assert.True(t, result)
	})

	t.Run("owner should have contributor group", func(t *testing.T) {
		result := HasGroup(models.GroupOwner, models.GroupContributor)
		assert.True(t, result, "Owner should satisfy Contributor requirement")
	})

	t.Run("owner should have viewer group", func(t *testing.T) {
		result := HasGroup(models.GroupOwner, models.GroupViewer)
		assert.True(t, result, "Owner should satisfy Viewer requirement")
	})

	t.Run("contributor should have contributor group", func(t *testing.T) {
		result := HasGroup(models.GroupContributor, models.GroupContributor)
		assert.True(t, result)
	})

	t.Run("contributor should have viewer group", func(t *testing.T) {
		result := HasGroup(models.GroupContributor, models.GroupViewer)
		assert.True(t, result, "Contributor should satisfy Viewer requirement")
	})

	t.Run("contributor should NOT have owner group", func(t *testing.T) {
		result := HasGroup(models.GroupContributor, models.GroupOwner)
		assert.False(t, result, "Contributor should NOT satisfy Owner requirement (privilege escalation)")
	})

	t.Run("viewer should have viewer group", func(t *testing.T) {
		result := HasGroup(models.GroupViewer, models.GroupViewer)
		assert.True(t, result)
	})

	t.Run("viewer should NOT have contributor group", func(t *testing.T) {
		result := HasGroup(models.GroupViewer, models.GroupContributor)
		assert.False(t, result, "Viewer should NOT satisfy Contributor requirement (privilege escalation)")
	})

	t.Run("viewer should NOT have owner group", func(t *testing.T) {
		result := HasGroup(models.GroupViewer, models.GroupOwner)
		assert.False(t, result, "Viewer should NOT satisfy Owner requirement (privilege escalation)")
	})
}

// TestHasGroup_EdgeCases tests edge cases and security scenarios.
func TestHasGroup_EdgeCases(t *testing.T) {
	t.Run("unknown group should not have any valid group", func(t *testing.T) {
		unknownGroup := models.Group("superuser")

		assert.False(t, HasGroup(unknownGroup, models.GroupOwner))
		assert.False(t, HasGroup(unknownGroup, models.GroupContributor))
		assert.False(t, HasGroup(unknownGroup, models.GroupViewer))
	})

	t.Run("valid group with unknown required group returns true due to rank comparison", func(t *testing.T) {
		unknownGroup := models.Group("admin")

		// NOTE: Current implementation returns true because unknown groups have rank 0
		// and Owner (rank 3) >= 0. This could be a security concern.
		assert.True(t, HasGroup(models.GroupOwner, unknownGroup),
			"Current implementation: Owner (rank 3) >= unknown (rank 0)")
	})

	t.Run("empty string group should not have any privileges", func(t *testing.T) {
		emptyGroup := models.Group("")

		assert.False(t, HasGroup(emptyGroup, models.GroupOwner))
		assert.False(t, HasGroup(emptyGroup, models.GroupContributor))
		assert.False(t, HasGroup(emptyGroup, models.GroupViewer))
	})

	t.Run("case sensitivity check", func(t *testing.T) {
		// Groups are case-sensitive, "Owner" != "owner"
		wrongCase := models.Group("Owner") // Should be "owner"

		assert.False(t, HasGroup(wrongCase, models.GroupViewer),
			"Case-sensitive group should not grant privileges")
	})
}

// TestHasGroup_TableDriven comprehensive test matrix.
func TestHasGroup_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		userGroup     models.Group
		requiredGroup models.Group
		expected      bool
		description   string
	}{
		// Owner scenarios
		{"Owner->Owner", models.GroupOwner, models.GroupOwner, true, "Owner can do Owner things"},
		{"Owner->Contributor", models.GroupOwner, models.GroupContributor, true, "Owner can do Contributor things"},
		{"Owner->Viewer", models.GroupOwner, models.GroupViewer, true, "Owner can do Viewer things"},

		// Contributor scenarios
		{"Contributor->Owner", models.GroupContributor, models.GroupOwner, false, "Contributor CANNOT do Owner things"},
		{
			"Contributor->Contributor",
			models.GroupContributor,
			models.GroupContributor,
			true,
			"Contributor can do Contributor things",
		},
		{"Contributor->Viewer", models.GroupContributor, models.GroupViewer, true, "Contributor can do Viewer things"},

		// Viewer scenarios
		{"Viewer->Owner", models.GroupViewer, models.GroupOwner, false, "Viewer CANNOT do Owner things"},
		{
			"Viewer->Contributor",
			models.GroupViewer,
			models.GroupContributor,
			false,
			"Viewer CANNOT do Contributor things",
		},
		{"Viewer->Viewer", models.GroupViewer, models.GroupViewer, true, "Viewer can do Viewer things"},

		// Edge cases
		{"Unknown->Owner", models.Group("unknown"), models.GroupOwner, false, "Unknown group has no privileges"},
		{"Owner->Unknown", models.GroupOwner, models.Group("unknown"), true, "Owner (rank 3) >= unknown (rank 0)"},
		{"Empty->Viewer", models.Group(""), models.GroupViewer, false, "Empty group has no privileges"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasGroup(tt.userGroup, tt.requiredGroup)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestHasGroup_SecurityImplications tests security-critical scenarios.
func TestHasGroup_SecurityImplications(t *testing.T) {
	t.Run("prevent horizontal privilege escalation", func(t *testing.T) {
		// A viewer trying to act as owner
		assert.False(t, HasGroup(models.GroupViewer, models.GroupOwner),
			"Must prevent viewer from gaining owner privileges")

		// A contributor trying to act as owner
		assert.False(t, HasGroup(models.GroupContributor, models.GroupOwner),
			"Must prevent contributor from gaining owner privileges")
	})

	t.Run("ensure proper permission downgrade", func(t *testing.T) {
		// Owner can safely downgrade to contributor
		assert.True(t, HasGroup(models.GroupOwner, models.GroupContributor))

		// Owner can safely downgrade to viewer
		assert.True(t, HasGroup(models.GroupOwner, models.GroupViewer))

		// Contributor can safely downgrade to viewer
		assert.True(t, HasGroup(models.GroupContributor, models.GroupViewer))
	})

	t.Run("unauthorized groups behavior", func(t *testing.T) {
		unauthorizedGroups := []models.Group{
			models.Group("superowner"),
			models.Group("admin"),
			models.Group("root"),
			models.Group("moderator"),
		}

		for _, unauthorizedGroup := range unauthorizedGroups {
			// Unauthorized groups shouldn't grant any privileges
			assert.False(t, HasGroup(unauthorizedGroup, models.GroupViewer),
				"Unauthorized group %s should not grant viewer access", unauthorizedGroup)

			// NOTE: Current implementation returns true because Owner (rank 3) >= unknown (rank 0)
			// This is documented behavior matching the role checker
			assert.True(t, HasGroup(models.GroupOwner, unauthorizedGroup),
				"Current implementation: Owner (rank 3) >= unknown group (rank 0)")
		}
	})
}
