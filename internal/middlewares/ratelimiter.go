package middlewares

import (
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func getClientIP(r *http.Request, trustedProxies []*net.IPNet) (string, error) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			remoteIP = r.RemoteAddr
		} else {
			return "", err
		}
	}

	if !isTrustedProxy(remoteIP, trustedProxies) {
		return remoteIP, nil
	}

	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor == "" {
		return remoteIP, nil
	}

	hops := strings.Split(xForwardedFor, ",")
	for _, hop := range slices.Backward(hops) {
		hop = strings.TrimSpace(hop)
		if hop != "" && !isTrustedProxy(hop, trustedProxies) {
			return hop, nil
		}
	}

	return remoteIP, nil
}

func isTrustedProxy(ip string, trustedProxies []*net.IPNet) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, network := range trustedProxies {
		if network.Contains(parsed) {
			return true
		}
	}
	return false
}

func applyRateLimit(
	next http.Handler,
	w http.ResponseWriter,
	r *http.Request,
	c cache.ICache,
	userIdentifier string,
	requestsPerMinute int,
) {
	retryAfter, err := cache.GetRateLimit(c, userIdentifier, requestsPerMinute)

	if err != nil {
		zap.L().Error("error", zap.Error(err))
		helpers.RespondWithError(w, 500, []string{apierrors.CodeInternalServerError})
		return
	}

	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		helpers.RespondWithError(w, http.StatusTooManyRequests, []string{apierrors.CodeRateLimitExceeded})
		return
	}

	next.ServeHTTP(w, r)
}

func RateLimit(
	cache cache.ICache,
	trustedProxies []*net.IPNet,
	authenticatedRequestsPerMinute int,
	unauthenticatedRequestsPerMinute int,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.RateLimit")
			defer span.End()
			r = r.WithContext(ctx)

			claims, err := helpers.GetUserClaims(r.Context())
			if err != nil {
				ipAddress, err2 := getClientIP(r, trustedProxies)
				if err2 != nil {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithErrorCtx(r.Context(), w, 500, []string{apierrors.CodeInternalServerError})
					return
				}
				applyRateLimit(next, w, r, cache, ipAddress, unauthenticatedRequestsPerMinute)
			} else {
				userID := claims.UserID.String()
				applyRateLimit(next, w, r, cache, userID, authenticatedRequestsPerMinute)
			}
		}
		return http.HandlerFunc(fn)
	}
}
