package middlewares

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func ClientInfo(trustedProxies []string) func(next http.Handler) http.Handler {
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

func getClientIP(r *http.Request, trustedProxies []string) (string, error) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			remoteIP = r.RemoteAddr
		} else {
			return "", err
		}
	}

	if len(trustedProxies) == 0 {
		return remoteIP, nil
	}

	if !slices.Contains(trustedProxies, remoteIP) {
		return remoteIP, nil
	}

	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0]), nil
		}
	}

	return remoteIP, nil
}
