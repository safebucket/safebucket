package middlewares

import (
	"api/internal/configuration"
	h "api/internal/helpers"
	"api/internal/models"
	"api/internal/rbac"
	"github.com/casbin/casbin/v2"
	"net/http"
)

func Authorize(e *casbin.Enforcer, resource rbac.Resource, action rbac.Action, objectIdIndex int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			domain := configuration.DefaultDomain

			var id string

			if objectIdIndex == -1 {
				id = configuration.NilUUID
			} else {
				ids, ok := h.ParseUUIDs(w, r)
				if !ok {
					h.RespondWithError(w, 401, []string{"Not Authorized"})
					return
				}
				id = ids[objectIdIndex].String()
			}

			authorized, err := e.Enforce(domain,
				r.Context().Value(models.UserClaimKey{}).(models.UserClaims).UserID.String(),
				resource.String(),
				id,
				action.String())

			if err != nil {
				h.RespondWithError(w, 401, []string{err.Error()})
				return
			}
			if !authorized {
				h.RespondWithError(w, 401, []string{"Not Authorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
