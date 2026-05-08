package handlers

import (
	"net/http"
	"strings"
	"time"
)

const (
	cookieAccessToken  = "safebucket_access_token"  //nolint:gosec // G101: not a credential, just a cookie name.
	cookieRefreshToken = "safebucket_refresh_token" //nolint:gosec // G101: not a credential, just a cookie name.
	cookieAuthProvider = "safebucket_auth_provider"
	cookieMFAToken     = "safebucket_mfa_token" //nolint:gosec // G101: not a credential, just a cookie name.

	mfaCookieMaxAgeSeconds = 15 * 60
)

func isSecureRequest(r *http.Request, forceSecure bool) bool {
	if forceSecure || r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.EqualFold(strings.TrimSpace(strings.SplitN(proto, ",", 2)[0]), "https")
	}
	return false
}

func SetAuthCookies(w http.ResponseWriter, r *http.Request, access, refresh, provider string, forceSecure bool) {
	expiration := time.Now().Add(365 * 24 * time.Hour)
	secure := isSecureRequest(r, forceSecure)

	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieAccessToken,
		Value:    access,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieRefreshToken,
		Value:    refresh,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieAuthProvider,
		Value:    provider,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	clearOneCookie(w, cookieMFAToken)
}

func SetAccessCookie(w http.ResponseWriter, r *http.Request, access string, forceSecure bool) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieAccessToken,
		Value:    access,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecureRequest(r, forceSecure),
	})
}

func SetMFACookie(w http.ResponseWriter, r *http.Request, token string, forceSecure bool) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieMFAToken,
		Value:    token,
		MaxAge:   mfaCookieMaxAgeSeconds,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecureRequest(r, forceSecure),
	})
	clearOneCookie(w, cookieAccessToken)
	clearOneCookie(w, cookieRefreshToken)
	clearOneCookie(w, cookieAuthProvider)
}

func ClearMFACookie(w http.ResponseWriter) {
	clearOneCookie(w, cookieMFAToken)
}

func ClearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{cookieAccessToken, cookieRefreshToken, cookieAuthProvider, cookieMFAToken} {
		clearOneCookie(w, name)
	}
}

func clearOneCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: clearing cookie, Secure not needed for expiry.
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
