package middlewares

import (
	"context"
	"net/http"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
)

type AuthExcludedKey struct{}

func Authenticate(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded from auth (default = auth required)
			excluded := isPathExcludedFromAuth(r.URL.Path, r.Method)
			ctx := context.WithValue(r.Context(), AuthExcludedKey{}, excluded)

			if excluded {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			accessToken := r.Header.Get("Authorization")

			// Parse token (signature, expiry validation only - no audience check)
			userClaims, err := helpers.ParseToken(jwtSecret, accessToken, true)
			if err != nil {
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
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

	return false // Auth required by default
}
