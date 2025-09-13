package middlewares

import (
	"api/internal/cache"
	"api/internal/helpers"
	"net"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const authenticatedRequestsPerMinute = 200
const unauthenticatedRequestsPerMinute = 20

func getClientIP(r *http.Request, trustedProxies []string) (string, error) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			remoteIP = r.RemoteAddr
		} else {
			return "", err
		}
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
	cache cache.ICache,
	userIdentifier string,
	requestsPerMinute int,
) {
	retryAfter, err := cache.GetRateLimit(userIdentifier, requestsPerMinute)
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

func RateLimit(cache cache.ICache, trustedProxies []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			claims, err := helpers.GetUserClaims(r.Context())
			if err != nil {
				ipAddress, err := getClientIP(r, trustedProxies)
				if err != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithError(w, 500, []string{"INTERNAL_SERVER_ERROR"})
					return
				}
				applyRateLimit(next, w, r, cache, ipAddress, unauthenticatedRequestsPerMinute)
			} else {
				userId := claims.UserID.String()
				applyRateLimit(next, w, r, cache, userId, authenticatedRequestsPerMinute)
			}
		}
		return http.HandlerFunc(fn)
	}
}
