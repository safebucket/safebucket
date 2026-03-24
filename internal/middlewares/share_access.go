package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"gorm.io/gorm"
)

type ShareKey struct{}

func ValidateShareAccess(db *gorm.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ids, ok := helpers.ParseUUIDs(w, r)
			if !ok {
				return
			}

			shareID := ids[0]

			var share models.Share
			if db.Where("id = ?", shareID).Find(&share).RowsAffected == 0 {
				helpers.RespondWithError(w, 404, []string{"SHARE_NOT_FOUND"})
				return
			}

			if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
				helpers.RespondWithError(w, 410, []string{"SHARE_EXPIRED"})
				return
			}

			if share.MaxViews != nil && share.CurrentViews >= *share.MaxViews {
				helpers.RespondWithError(w, 403, []string{"SHARE_MAX_VIEWS_REACHED"})
				return
			}

			ctx := context.WithValue(r.Context(), ShareKey{}, share)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValidateShareToken(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			share, _ := r.Context().Value(ShareKey{}).(models.Share)

			if share.HashedPassword == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				helpers.RespondWithError(w, 401, []string{"SHARE_TOKEN_REQUIRED"})
				return
			}

			claims, err := helpers.ParseShareToken(jwtSecret, authHeader)
			if err != nil || claims.ShareID != share.ID {
				helpers.RespondWithError(w, 401, []string{"SHARE_TOKEN_INVALID"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
