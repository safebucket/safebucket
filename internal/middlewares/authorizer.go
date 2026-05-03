package middlewares

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/tracing"

	"gorm.io/gorm"
)

// AuthorizeRole checks if the authenticated user has at least the required role
// Uses hierarchical role checking (Admin > User > Guest).
func AuthorizeRole(requiredRole models.Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.AuthorizeRole")
			defer span.End()
			r = r.WithContext(ctx)

			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			if !rbac.HasRole(userClaims.Role, requiredRole) {
				h.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
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
			ctx, span := tracing.StartSpan(r.Context(), "middleware.AuthorizeGroup")
			defer span.End()
			r = r.WithContext(ctx)

			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			if userClaims.Role == models.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			ids, ok := h.ParseUUIDs(w, r)
			if !ok {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			if bucketIDIndex >= len(ids) {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			bucketID := ids[bucketIDIndex]

			hasAccess, err := rbac.HasBucketAccess(db, userClaims.UserID, bucketID, requiredGroup)
			if err != nil {
				h.RespondWithErrorCtx(r.Context(), w, 500, []string{apierrors.CodeInternalServerError})
				return
			}

			if !hasAccess {
				h.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthorizeSelfOrAdmin(targetUserIDIndex int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.AuthorizeSelfOrAdmin")
			defer span.End()
			r = r.WithContext(ctx)

			userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
			if !ok {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			ids, ok := h.ParseUUIDs(w, r)
			if !ok {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			if targetUserIDIndex >= len(ids) {
				h.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeUnauthorized})
				return
			}

			targetUserID := ids[targetUserIDIndex]

			if userClaims.UserID == targetUserID {
				next.ServeHTTP(w, r)
				return
			}

			if userClaims.Role != models.RoleAdmin {
				h.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
