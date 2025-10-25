package middlewares

import (
	h "api/internal/helpers"
	"api/internal/models"
	"api/internal/rbac"
	"net/http"

	"gorm.io/gorm"
)

// AuthorizeRole checks if the authenticated user has at least the required role
// Uses hierarchical role checking (Admin > User > Guest)
func AuthorizeRole(requiredRole models.Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			if !rbac.HasRole(userClaims.Role, requiredRole) {
				h.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthorizeGroup checks if the authenticated user has at least the required group access to a bucket
// Uses hierarchical group checking (Owner > Contributor > Viewer)
// The bucketIdIndex parameter specifies which URL parameter contains the bucket ID
func AuthorizeGroup(db *gorm.DB, requiredGroup models.Group, bucketIdIndex int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			ids, ok := h.ParseUUIDs(w, r)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			if bucketIdIndex >= len(ids) {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			bucketID := ids[bucketIdIndex]

			hasAccess, err := rbac.HasBucketAccess(db, userClaims.UserID, bucketID, requiredGroup)
			if err != nil {
				h.RespondWithError(w, 500, []string{"INTERNAL_SERVER_ERROR"})
				return
			}

			if !hasAccess {
				h.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AuthorizeSelfOrAdmin allows the request if either:
// 1. The authenticated user is accessing their own resource (user ID matches target ID in URL)
// 2. The authenticated user has Admin role
// The targetUserIdIndex parameter specifies which URL parameter contains the target user ID
func AuthorizeSelfOrAdmin(targetUserIdIndex int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			ids, ok := h.ParseUUIDs(w, r)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			if targetUserIdIndex >= len(ids) {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			targetUserID := ids[targetUserIdIndex]

			if userClaims.UserID == targetUserID {
				next.ServeHTTP(w, r)
				return
			}

			if userClaims.Role != models.RoleAdmin {
				h.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
