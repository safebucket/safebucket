package helpers

import (
	"api/internal/models"
	"net/http"
)

func Authorize(jwtConf models.JWTConfiguration, fu func(*http.Request, *models.UserClaims) error) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			accessToken := r.Header.Get("Authorization")
			token, err := ParseAccessToken(jwtConf.Secret, accessToken)

			if err != nil {
				RespondWithError(w, 401, []string{err.Error()})
			}

			err = fu(r, token)

			if err != nil {
				RespondWithError(w, 401, []string{err.Error()})
			}

			next.ServeHTTP(w, r)
		})
	}
}
