package middlewares

import (
	"api/internal/core"
	"api/internal/helpers"
	"api/internal/models"
	"go.uber.org/zap"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const authenticatedRequestsPerSecond = 200
const unauthenticatedRequestsPerSecond = 20

func applyRateLimit(
	next http.Handler,
	w http.ResponseWriter,
	r *http.Request,
	cache core.Cache,
	userIdentifier string,
	requestsPerSecond int,
) {
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

	next.ServeHTTP(w, r)
}

func RateLimit(cache core.Cache, trustedProxies []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				ipAddress, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
				}

				if len(trustedProxies) > 0 {
					xForwardedFor := r.Header.Get("X-Forwarded-For")
					ips := strings.Split(xForwardedFor, ",")
					if xForwardedFor != "" && len(ips) == 0 {
						ipAddress = strings.TrimSpace(ips[0])
					}
				} else {
					ipAddress, _, err = net.SplitHostPort(r.RemoteAddr)
					if err != nil {
						zap.L().Error("error", zap.Error(err))
						helpers.RespondWithError(w, 500, []string{"Unexpected Server Error"})
					}
				}

				applyRateLimit(next, w, r, cache, ipAddress, unauthenticatedRequestsPerSecond)
			} else {
				userId := r.Context().Value(models.UserClaimKey{}).(models.UserClaims).UserID.String()

				applyRateLimit(next, w, r, cache, userId, authenticatedRequestsPerSecond)
			}
		}
		return http.HandlerFunc(fn)
	}
}
