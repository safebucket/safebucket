package middlewares

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
)

func CSRFGuard(allowedOrigins []string) func(next http.Handler) http.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[strings.ToLower(strings.TrimRight(o, "/"))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isStateChanging(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if !hasAuthCookie(r) {
				next.ServeHTTP(w, r)
				return
			}

			if allowAll {
				next.ServeHTTP(w, r)
				return
			}

			origin := requestOrigin(r)
			if origin == "" {
				helpers.RespondWithErrorCtx(r.Context(), w, http.StatusForbidden,
					[]string{apierrors.CodeForbidden})
				return
			}

			if originMatchesHost(origin, r) {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := allowed[strings.ToLower(origin)]; ok {
				next.ServeHTTP(w, r)
				return
			}

			helpers.RespondWithErrorCtx(r.Context(), w, http.StatusForbidden,
				[]string{apierrors.CodeForbidden})
		})
	}
}

func hasAuthCookie(r *http.Request) bool {
	for _, name := range []string{
		configuration.CookieAccessToken,
		configuration.CookieRefreshToken,
		configuration.CookieMFAToken,
		configuration.CookieShareToken,
	} {
		if _, err := r.Cookie(name); err == nil {
			return true
		}
	}
	return false
}

func isStateChanging(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requestOrigin(r *http.Request) string {
	if o := r.Header.Get("Origin"); o != "" && o != "null" {
		return strings.TrimRight(o, "/")
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
			return strings.TrimRight(u.Scheme+"://"+u.Host, "/")
		}
	}
	return ""
}

func originMatchesHost(origin string, r *http.Request) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if !strings.EqualFold(u.Host, r.Host) {
		return false
	}
	if r.TLS != nil && !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	return true
}
