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
	authCookieDuration     = 365 * 24 * time.Hour
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

func longLivedCookie(name, value string, secure bool) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(authCookieDuration),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	}
}

func clearedCookie(name string) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // G124: clearing cookie, Secure not needed for expiry.
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func BuildAuthCookies(isSecure bool, access, refresh, provider string) []*http.Cookie {
	return []*http.Cookie{
		longLivedCookie(cookieAccessToken, access, isSecure),
		longLivedCookie(cookieRefreshToken, refresh, isSecure),
		longLivedCookie(cookieAuthProvider, provider, isSecure),
		clearedCookie(cookieMFAToken),
	}
}

func BuildAccessCookie(isSecure bool, access string) []*http.Cookie {
	return []*http.Cookie{longLivedCookie(cookieAccessToken, access, isSecure)}
}

func BuildMFACookie(isSecure bool, token string) []*http.Cookie {
	mfa := &http.Cookie{ //nolint:gosec // G124: Secure is set conditionally based on TLS/forceSecure.
		Name:     cookieMFAToken,
		Value:    token,
		MaxAge:   mfaCookieMaxAgeSeconds,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   isSecure,
	}
	return []*http.Cookie{
		mfa,
		clearedCookie(cookieAccessToken),
		clearedCookie(cookieRefreshToken),
		clearedCookie(cookieAuthProvider),
	}
}

func BuildClearAuthCookies() []*http.Cookie {
	return []*http.Cookie{
		clearedCookie(cookieAccessToken),
		clearedCookie(cookieRefreshToken),
		clearedCookie(cookieAuthProvider),
		clearedCookie(cookieMFAToken),
	}
}

func writeCookies(w http.ResponseWriter, cookies []*http.Cookie) {
	for _, c := range cookies {
		http.SetCookie(w, c)
	}
}

func SetAuthCookies(w http.ResponseWriter, r *http.Request, access, refresh, provider string, forceSecure bool) {
	writeCookies(w, BuildAuthCookies(isSecureRequest(r, forceSecure), access, refresh, provider))
}

func SetAccessCookie(w http.ResponseWriter, r *http.Request, access string, forceSecure bool) {
	writeCookies(w, BuildAccessCookie(isSecureRequest(r, forceSecure), access))
}

func SetMFACookie(w http.ResponseWriter, r *http.Request, token string, forceSecure bool) {
	writeCookies(w, BuildMFACookie(isSecureRequest(r, forceSecure), token))
}

func ClearMFACookie(w http.ResponseWriter) {
	http.SetCookie(w, clearedCookie(cookieMFAToken))
}

func ClearAuthCookies(w http.ResponseWriter) {
	writeCookies(w, BuildClearAuthCookies())
}
