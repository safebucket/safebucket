package middlewares

import (
	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"github.com/casbin/casbin/v2"
	"net/http"
)

func Authorize(op models.Operation, e *casbin.Enforcer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			domain := configuration.DefaultDomain
			authorized, err := e.Enforce(domain,
				r.Context().Value(configuration.ContextUserClaimKey).(*models.UserClaims).UserID.String(),
				op.Object,
				op.ObjectID,
				op.Action)

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
