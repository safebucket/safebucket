package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func cookieByName(t *testing.T, rr *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSetAuthCookies_AllAttributes(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.TLS = &tls.ConnectionState{}

	SetAuthCookies(rr, req, "atok", "rtok", "local", false)

	for _, name := range []string{cookieAccessToken, cookieRefreshToken, cookieAuthProvider} {
		c := cookieByName(t, rr, name)
		if c == nil {
			t.Fatalf("missing cookie %q", name)
		}
		if !c.HttpOnly {
			t.Errorf("%s: expected HttpOnly", name)
		}
		if c.SameSite != http.SameSiteStrictMode {
			t.Errorf("%s: expected SameSite=Strict, got %v", name, c.SameSite)
		}
		if !c.Secure {
			t.Errorf("%s: expected Secure under TLS", name)
		}
		if c.Path != "/" {
			t.Errorf("%s: expected path /, got %s", name, c.Path)
		}
	}
}

func TestSetAuthCookies_SecureOffWithoutTLS(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)

	SetAuthCookies(rr, req, "a", "r", "local", false)

	c := cookieByName(t, rr, cookieAccessToken)
	if c == nil {
		t.Fatalf("missing access cookie")
	}
	if c.Secure {
		t.Errorf("expected Secure=false on plain HTTP request")
	}
}

func TestSetAuthCookies_ForceSecure(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)

	SetAuthCookies(rr, req, "a", "r", "local", true)

	if c := cookieByName(t, rr, cookieAccessToken); c == nil || !c.Secure {
		t.Fatalf("expected forced Secure cookie")
	}
}

func TestSetAuthCookies_XForwardedProtoHTTPS(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	SetAuthCookies(rr, req, "a", "r", "local", false)

	if c := cookieByName(t, rr, cookieAccessToken); c == nil || !c.Secure {
		t.Fatalf("expected Secure when X-Forwarded-Proto=https")
	}
}

func TestSetAuthCookies_XForwardedProtoHTTPSWithList(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")

	SetAuthCookies(rr, req, "a", "r", "local", false)

	if c := cookieByName(t, rr, cookieAccessToken); c == nil || !c.Secure {
		t.Fatalf("expected Secure with list-style X-Forwarded-Proto")
	}
}

func TestSetAuthCookies_XForwardedProtoHTTPNoSecure(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.Header.Set("X-Forwarded-Proto", "http")

	SetAuthCookies(rr, req, "a", "r", "local", false)

	if c := cookieByName(t, rr, cookieAccessToken); c == nil || c.Secure {
		t.Fatalf("expected Secure=false when forwarded proto is http")
	}
}

func TestClearAuthCookies(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearAuthCookies(rr)

	for _, name := range []string{cookieAccessToken, cookieRefreshToken, cookieAuthProvider, cookieMFAToken} {
		c := cookieByName(t, rr, name)
		if c == nil {
			t.Fatalf("missing cleared cookie %q", name)
		}
		if c.MaxAge >= 0 {
			t.Errorf("%s: expected MaxAge<0 to clear, got %d", name, c.MaxAge)
		}
		if c.Value != "" {
			t.Errorf("%s: expected empty value, got %q", name, c.Value)
		}
	}
}

func TestSetAuthCookies_AlsoClearsMFACookie(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)

	SetAuthCookies(rr, req, "a", "r", "local", false)

	mfa := cookieByName(t, rr, cookieMFAToken)
	if mfa == nil {
		t.Fatalf("expected MFA cookie clear directive")
	}
	if mfa.MaxAge >= 0 {
		t.Errorf("expected MFA cookie MaxAge<0, got %d", mfa.MaxAge)
	}
	if mfa.Value != "" {
		t.Errorf("expected empty MFA cookie value, got %q", mfa.Value)
	}
}

func TestSetMFACookie_SetsRestrictedAndClearsFullAuth(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/whatever", nil)
	req.TLS = &tls.ConnectionState{}

	SetMFACookie(rr, req, "mfa-tok", false)

	c := cookieByName(t, rr, cookieMFAToken)
	if c == nil {
		t.Fatalf("missing MFA cookie")
	}
	if c.Value != "mfa-tok" {
		t.Errorf("expected MFA cookie value, got %q", c.Value)
	}
	if !c.HttpOnly {
		t.Errorf("expected HttpOnly")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSite=Strict, got %v", c.SameSite)
	}
	if !c.Secure {
		t.Errorf("expected Secure under TLS")
	}
	if c.MaxAge != mfaCookieMaxAgeSeconds {
		t.Errorf("expected MaxAge=%d, got %d", mfaCookieMaxAgeSeconds, c.MaxAge)
	}

	for _, name := range []string{cookieAccessToken, cookieRefreshToken, cookieAuthProvider} {
		cleared := cookieByName(t, rr, name)
		if cleared == nil {
			t.Fatalf("expected %s clear directive", name)
		}
		if cleared.MaxAge >= 0 {
			t.Errorf("%s: expected MaxAge<0, got %d", name, cleared.MaxAge)
		}
	}
}

func TestClearMFACookie(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearMFACookie(rr)

	c := cookieByName(t, rr, cookieMFAToken)
	if c == nil {
		t.Fatalf("missing cleared MFA cookie")
	}
	if c.MaxAge >= 0 {
		t.Errorf("expected MaxAge<0, got %d", c.MaxAge)
	}
	if c.Value != "" {
		t.Errorf("expected empty value, got %q", c.Value)
	}
}
