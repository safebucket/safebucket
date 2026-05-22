package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
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
				helpers.RespondWithError(w, http.StatusNotFound, []string{apierrors.CodeShareNotFound})
				return
			}

			if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
				helpers.RespondWithError(w, http.StatusGone, []string{apierrors.CodeShareExpired})
				return
			}

			if share.MaxViews != nil && share.CurrentViews >= *share.MaxViews {
				helpers.RespondWithError(w, http.StatusForbidden, []string{apierrors.CodeShareMaxViewsReached})
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

			cookie, err := r.Cookie(configuration.CookieShareToken)
			if err != nil {
				helpers.RespondWithError(w, http.StatusUnauthorized, []string{apierrors.CodeShareTokenRequired})
				return
			}

			claims, err := helpers.ParseShareToken(jwtSecret, cookie.Value)
			if err != nil || claims.ShareID != share.ID {
				helpers.RespondWithError(w, http.StatusUnauthorized, []string{apierrors.CodeShareTokenInvalid})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
