package middlewares

import (
	"context"
	"net/http"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
)

// AuthExcludedKey is used to store auth exclusion flag in context.
type AuthExcludedKey struct{}

// Authenticate middleware handles authentication.
func Authenticate(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded from auth (default = auth required)
			excluded := isAuthExcluded(r.URL.Path, r.Method)
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

// isAuthExcluded checks if path is excluded from authentication.
// Returns false by default (auth required unless explicitly excluded).
func isAuthExcluded(path, method string) bool {
	// Phase 1: Fast exact path match (O(1) lookup)
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
