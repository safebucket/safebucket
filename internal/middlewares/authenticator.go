package middlewares

import (
	"context"
	"net/http"
	"strings"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
)

func Authenticate(jwtSecret string, mfaRequired bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			accessToken := r.Header.Get("Authorization")
			userClaims, err := helpers.ParseAccessToken(jwtSecret, accessToken)
			if err != nil {
				if isMFABypassPath(r.URL.Path, r.Method) {
					next.ServeHTTP(w, r)
					return
				}
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			if mfaRequired && !userClaims.MFA && userClaims.Provider == string(models.LocalProviderType) {
				if !isMFABypassPath(r.URL.Path, r.Method) {
					helpers.RespondWithError(w, 403, []string{"MFA_SETUP_REQUIRED"})
					return
				}
			}

			ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func isMFABypassPath(path, method string) bool {
	for _, rule := range configuration.MFABypassRules {
		if rule.Method != "*" && rule.Method != method {
			continue
		}

		if rule.PathSuffix == "" {
			if strings.HasPrefix(path, rule.PathPrefix) {
				remaining := strings.TrimPrefix(path, rule.PathPrefix)
				if remaining == "" || !strings.Contains(remaining, "/") {
					return true
				}
			}
		} else {
			if strings.HasPrefix(path, rule.PathPrefix) && strings.HasSuffix(path, rule.PathSuffix) {
				return true
			}
		}
	}
	return false
}

func isExcluded(path, method string) bool {
	if exactRules, exists := configuration.AuthRuleExactMatchPath[path]; exists {
		for _, rule := range exactRules {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth
			}
		}
	}

	for _, rule := range configuration.AuthRulePrefixMatchPath {
		if strings.HasPrefix(path, rule.Path) {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth
			}
		}
	}

	return false
}
