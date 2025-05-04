package middlewares

import (
	"api/internal/configuration"
	h "api/internal/helpers"
	"api/internal/models"
	"github.com/casbin/casbin/v2"
	"net/http"
)

func Authorize(authzParameters models.AuthzParameters, e *casbin.Enforcer) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			domain := configuration.DefaultDomain

			var id string

			if authzParameters.ObjectIdIndex == -1 {
				id = configuration.NilUUID
			} else {
				ids, ok := h.ParseUUIDs(w, r)
				if !ok {
					h.RespondWithError(w, 401, []string{"Not Authorized"})
					return
				}
				id = ids[authzParameters.ObjectIdIndex].String()
			}

			op := models.Operation{
				Object:   authzParameters.ObjectType.String(),
				ObjectID: id,
				Action:   authzParameters.Action.String(),
			}

			authorized, err := e.Enforce(domain,
				r.Context().Value(configuration.ContextUserClaimKey).(*models.UserClaims).UserID.String(),
				op.Object,
				op.ObjectID,
				op.Action)

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
