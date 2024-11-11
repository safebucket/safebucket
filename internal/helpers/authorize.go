package helpers

import (
	"api/internal/models"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

// claims *models.UserClaims

func Authorize(jwtConf models.JWTConfiguration, DB *gorm.DB, e *casbin.Enforcer, object string, objectID string, operation string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Validate Token
			accessToken := r.Header.Get("Authorization")
			zap.L().Info(accessToken)

			//token, _ := ParseAccessToken(jwtConf.Secret, strings.TrimPrefix(accessToken, "Bearer "))
			//if err != nil {
			//	RespondWithError(w, 401, []string{err.Error()})
			//}
			//zap.L().Info(token.Email)

			searchUser := models.User{Email: "test5@gmail.com"}
			result := DB.Where("email = ?", searchUser.Email).First(&searchUser)
			if result.RowsAffected == 1 {
				data, _ := e.Enforce(&searchUser.ID, object, objectID, operation)
				if data {
					next.ServeHTTP(w, r)
				} else {
					RespondWithError(w, 403, []string{object + ":" + operation})
					return
				}
			}
			//

			//accessToken := r.Header.Get("Authorization")
			//token, err := h.ParseAccessToken(jwtConf.Secret, accessToken)
			//if err != nil {
			//	h.RespondWithError(w, 401, []string{err.Error()})
			//}
			//err =
			//if err != nil {
			//	h.RespondWithError(w, 401, []string{err.Error()})
			//}
			next.ServeHTTP(w, r)
		})
	}
}
