package middlewares

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func ClientInfo(rawProxies []string) func(next http.Handler) http.Handler {
	trustedProxies := configuration.ParseTrustedProxies(rawProxies)
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.ClientInfo")
			defer span.End()
			r = r.WithContext(ctx)

			ip, err := getClientIP(r, trustedProxies)
			if err != nil {
				GetLogger(r).Warn("failed to resolve client IP", zap.Error(err))
			}

			info := models.ClientInfo{
				IP:        ip,
				UserAgent: r.UserAgent(),
			}

			ctx = context.WithValue(r.Context(), models.ClientInfoKey{}, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

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
		if hop == "" || net.ParseIP(hop) == nil {
			continue
		}
		if !isTrustedProxy(hop, trustedProxies) {
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
