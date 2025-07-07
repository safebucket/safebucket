package middlewares

import (
	"api/internal/core"
	"api/internal/helpers"
	"api/internal/models"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strconv"
)

const AuthenticatedRequestsPerSecond = 200
const UnauthenticatedRequestsPerSecond = 20

func RateLimit(cache core.Cache) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
					return
				}

				retryAfter, err := cache.GetRateLimit(ip, UnauthenticatedRequestsPerSecond)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
					return
				}

				if retryAfter > 0 {
					r.Header.Set("Retry-After", strconv.Itoa(retryAfter))
					helpers.RespondWithError(w, 429, []string{"Rate Limit Exceeded"})
					return
				}
			} else {
				user := r.Context().Value(models.UserClaimKey{}).(models.UserClaims).UserID.String()

				retryAfter, err := cache.GetRateLimit(user, AuthenticatedRequestsPerSecond)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
					return
				}

				if retryAfter > 0 {
					r.Header.Set("Retry-After", strconv.Itoa(retryAfter))
					helpers.RespondWithError(w, 429, []string{"Rate Limit Exceeded"})
					return
				}
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
