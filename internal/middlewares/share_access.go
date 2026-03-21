package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
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
			password := r.Header.Get("X-Share-Password")

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

			if share.HashedPassword != "" {
				if password == "" {
					helpers.RespondWithError(w, 401, []string{"SHARE_PASSWORD_REQUIRED"})
					return
				}
				match, err := argon2id.ComparePasswordAndHash(password, share.HashedPassword)
				if err != nil || !match {
					helpers.RespondWithError(w, 401, []string{"SHARE_PASSWORD_INVALID"})
					return
				}
			}

			ctx := context.WithValue(r.Context(), ShareKey{}, share)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
