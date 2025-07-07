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

const authenticatedRequestsPerSecond = 200
const unauthenticatedRequestsPerSecond = 20

func applyRateLimit(w http.ResponseWriter, r *http.Request, cache core.Cache, userIdentifier string, requestsPerSecond int) {
	retryAfter, err := cache.GetRateLimit(userIdentifier, requestsPerSecond)
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

func RateLimit(cache core.Cache) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				ipAddress, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
				}

				applyRateLimit(w, r, cache, ipAddress, unauthenticatedRequestsPerSecond)
			} else {
				userId := r.Context().Value(models.UserClaimKey{}).(models.UserClaims).UserID.String()

				applyRateLimit(w, r, cache, userId, authenticatedRequestsPerSecond)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
