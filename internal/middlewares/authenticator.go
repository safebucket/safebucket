package middlewares

import (
	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"context"
	"net/http"
	"strings"
)

func Authenticate(jwtConf models.JWTConfiguration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				next.ServeHTTP(w, r)
			} else {
				accessToken := r.Header.Get("Authorization")
				userClaims, err := helpers.ParseAccessToken(jwtConf.Secret, accessToken)
				if err != nil {
					helpers.RespondWithError(w, 403, []string{err.Error()})
					return
				}
				ctx := context.WithValue(r.Context(), configuration.ContextUserClaimKey, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
			}
		}
		return http.HandlerFunc(fn)
	}
}

func isExcluded(path string) bool {
	excludedPaths := configuration.ExcludedAuthPaths
	for _, value := range excludedPaths {
		if strings.HasPrefix(path, value) {
			return true
		}
	}
	return false
}
