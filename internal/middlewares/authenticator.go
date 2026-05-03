package middlewares

import (
	"context"
	"net/http"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"
)

type AuthExcludedKey struct{}

func Authenticate(
	jwtSecret string, c cache.ICache, refreshTokenExpiry int,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.Authenticate")
			defer span.End()
			r = r.WithContext(ctx)

			excluded := isPathExcludedFromAuth(r.URL.Path, r.Method)
			ctx = context.WithValue(r.Context(), AuthExcludedKey{}, excluded)

			if excluded {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			accessToken := r.Header.Get("Authorization")

			userClaims, err := helpers.ParseToken(jwtSecret, accessToken, true)
			if err != nil {
				helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}

			if userClaims.Audience[0] == configuration.AudienceAccessToken {
				if userClaims.SID == "" {
					helpers.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeSessionRevoked})
					return
				}

				maxAge := time.Duration(refreshTokenExpiry) * time.Minute
				active, sessionErr := cache.IsSessionActive(
					c, userClaims.UserID.String(), userClaims.SID, maxAge,
				)
				if sessionErr != nil || !active {
					helpers.RespondWithErrorCtx(r.Context(), w, 401, []string{apierrors.CodeSessionRevoked})
					return
				}
			}

			ctx = context.WithValue(ctx, models.UserClaimKey{}, userClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func isPathExcludedFromAuth(path, method string) bool {
	if m, ok := configuration.AuthExcludedExactPaths[path]; ok {
		if m == "*" || m == method {
			return true
		}
	}

	for _, rule := range configuration.AuthExcludedPatterns {
		if rule.Pattern.MatchString(path) && (rule.Method == "*" || rule.Method == method) {
			return true
		}
	}

	return false
}
