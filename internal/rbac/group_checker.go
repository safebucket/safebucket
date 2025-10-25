package rbac

import "api/internal/models"

// groupRank returns the hierarchical rank of a bucket group
// Higher rank means more permissions (Owner > Contributor > Viewer)
func groupRank(group models.Group) int {
	switch group {
	case models.GroupOwner:
		return 3
	case models.GroupContributor:
		return 2
	case models.GroupViewer:
		return 1
	default:
		return 0
	}
}

// HasGroup checks if a user's group meets or exceeds the required group
// Examples:
//   - Owner has Contributor access: true (Owner >= Contributor)
//   - Contributor has Owner access: false (Contributor < Owner)
//   - Contributor has Viewer access: true (Contributor >= Viewer)
func HasGroup(userGroup models.Group, requiredGroup models.Group) bool {
	return groupRank(userGroup) >= groupRank(requiredGroup)
}
