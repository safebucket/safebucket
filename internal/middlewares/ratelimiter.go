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

const authenticatedRequestsPerMinute = 200
const unauthenticatedRequestsPerMinute = 20

func getClientIP(r *http.Request, trustedProxies []string) (string, error) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	// If no trusted proxies configured, use remote address directly
	if len(trustedProxies) == 0 {
		return remoteIP, nil
	}

	// Verify the request is coming from a trusted proxy
	isTrustedProxy := false
	for _, proxy := range trustedProxies {
		if remoteIP == proxy {
			isTrustedProxy = true
			break
		}
	}

	// If not from trusted proxy, use remote address
	if !isTrustedProxy {
		return remoteIP, nil
	}

	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0]), nil
		}
	}

	// Fallback to remote address
	return remoteIP, nil
}

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
		helpers.RespondWithError(w, 500, []string{"INTERNAL_SERVER_ERROR"})
		return
	}

	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		helpers.RespondWithError(w, 429, []string{"RATE_LIMIT_EXCEEDED"})
		return
	}

	next.ServeHTTP(w, r)
}

func RateLimit(cache core.Cache, trustedProxies []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path, r.Method) {
				ipAddress, err := getClientIP(r, trustedProxies)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"INTERNAL_SERVER_ERROR"})
					return
				}

				applyRateLimit(next, w, r, cache, ipAddress, unauthenticatedRequestsPerMinute)
			} else {
				userId := r.Context().Value(models.UserClaimKey{}).(models.UserClaims).UserID.String()
				applyRateLimit(next, w, r, cache, userId, authenticatedRequestsPerMinute)
			}
		}
		return http.HandlerFunc(fn)
	}
}
