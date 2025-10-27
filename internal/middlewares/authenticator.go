package middlewares

import (
	"context"
	"net/http"
	"strings"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
)

func Authenticate(jwtSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
			} else {
				accessToken := r.Header.Get("Authorization")

				userClaims, err := helpers.ParseAccessToken(jwtSecret, accessToken)
				if err != nil {
					helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
					return
				}
				ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		}
		return http.HandlerFunc(fn)
	}
}

func isExcluded(path, method string) bool {
	// First check prefix matches for exclusions
	if exactRules, exists := configuration.AuthRuleExactMatchPath[path]; exists {
		for _, rule := range exactRules {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth // If RequireAuth is true, don't exclude (return false)
			}
		}
	}

	for _, rule := range configuration.AuthRulePrefixMatchPath {
		if strings.HasPrefix(path, rule.Path) {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth // If RequireAuth is true, don't exclude (return false)
			}
		}
	}

	// Default: require authentication (not excluded)
	return false
}
