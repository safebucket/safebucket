package middlewares

import (
	"context"
	"net/http"
	"strings"

	h "api/internal/helpers"
	"api/internal/models"
	"api/internal/rbac"

	"gorm.io/gorm"
)

// AuthorizeRole checks if the authenticated user has at least the required role
// Uses hierarchical role checking (Admin > User > Guest).
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
// The bucketIdIndex parameter specifies which URL parameter contains the bucket ID.
func AuthorizeGroup(
	db *gorm.DB,
	requiredGroup models.Group,
	bucketIDIndex int,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			if userClaims.Role == models.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			ids, ok := h.ParseUUIDs(w, r)
			if !ok {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			if bucketIDIndex >= len(ids) {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			bucketID := ids[bucketIDIndex]

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
// The targetUserIdIndex parameter specifies which URL parameter contains the target user ID.
func AuthorizeSelfOrAdmin(targetUserIDIndex int) func(next http.Handler) http.Handler {
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

			if targetUserIDIndex >= len(ids) {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			targetUserID := ids[targetUserIDIndex]

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

// MFAAuthorize handles authorization for MFA setup endpoints.
// It accepts either a standard Access Token (for already authenticated users)
// OR an MFA Token (for users in the login flow who need to set up MFA).
func MFAAuthorize(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				h.RespondWithError(w, 401, []string{"UNAUTHORIZED"})
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			// 1. Try parsing as Access Token (ParseAccessToken expects "Bearer " prefix)
			userClaims, err := h.ParseAccessToken(jwtSecret, authHeader)
			if err == nil {
				ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Try parsing as MFA Token
			userClaims, err = h.ParseMFAToken(jwtSecret, token)
			if err == nil {
				ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			h.RespondWithError(w, 403, []string{"FORBIDDEN"})
		}
		return http.HandlerFunc(fn)
	}
}
