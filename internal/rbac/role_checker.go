package rbac

import "api/internal/models"

// roleRank returns the hierarchical rank of a role
// Higher rank means more permissions (Admin > User > Guest)
func roleRank(role models.Role) int {
	switch role {
	case models.RoleAdmin:
		return 3
	case models.RoleUser:
		return 2
	case models.RoleGuest:
		return 1
	default:
		return 0
	}
}

// HasRole checks if a user's role meets or exceeds the required role
// Examples:
//   - Admin has User role: true (Admin >= User)
//   - User has Admin role: false (User < Admin)
//   - User has Guest role: true (User >= Guest)
func HasRole(userRole models.Role, requiredRole models.Role) bool {
	return roleRank(userRole) >= roleRank(requiredRole)
}
