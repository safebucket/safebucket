package rbac

import (
	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"context"
	"github.com/casbin/casbin/v2"
	"net/http"
)

func Authenticate(jwtConf models.JWTConfiguration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			accessToken := r.Header.Get("Authorization")
			userClaims, err := helpers.ParseAccessToken(jwtConf.Secret, accessToken)

			if err != nil {
				helpers.RespondWithError(w, 401, []string{err.Error()})
				return
			}

			key := "userclaims"
			ctx := context.WithValue(r.Context(), key, userClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Authorize(op models.Operation, e *casbin.Enforcer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			domain := configuration.DefaultDomain
			authorized, err := e.Enforce(domain, r.Context().Value("userclaims").(*models.UserClaims).UserID.String(), op.Object, op.ObjectId, op.Action)
			if err != nil {
				helpers.RespondWithError(w, 401, []string{err.Error()})
				return
			}

			if !authorized {
				helpers.RespondWithError(w, 401, []string{"Not Authorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
