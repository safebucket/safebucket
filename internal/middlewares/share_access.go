package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShareKey struct{}

func ValidateShareAccess(db *gorm.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			shareID := chi.URLParam(r, "shareId")
			if !validateShareID(shareID) {
				helpers.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeInvalidValue})
				return
			}

			var share models.Share
			var query *gorm.DB
			if id, err := uuid.Parse(shareID); err == nil && id.String() == shareID {
				query = db.Where("id = ? AND custom_id IS NULL", id)
			} else {
				query = db.Where("custom_id = ?", shareID)
			}

			result := query.Find(&share)
			if result.Error != nil {
				helpers.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
				return
			}
			if result.RowsAffected == 0 {
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
