package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newPassthroughCSRF(t *testing.T, allowed []string) http.Handler {
	t.Helper()
	mw := CSRFGuard(allowed)
	return mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestCSRFGuard_AllowsSafeMethods(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	req.Header.Set("Origin", "https://attacker.example")
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected GET to pass, got %d", rr.Code)
	}
}

func TestCSRFGuard_AllowsBearerOnlyClients(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bearer-only POST to pass, got %d", rr.Code)
	}
}

func TestCSRFGuard_RejectsForeignOrigin(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://attacker.example")
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestCSRFGuard_AcceptsAllowedOrigin(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)
	req.Host = "api.internal"
	req.Header.Set("Origin", "https://example.com")
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestCSRFGuard_AcceptsSameOrigin(t *testing.T) {
	h := newPassthroughCSRF(t, []string{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)
	req.Host = "app.local"
	req.Header.Set("Origin", "http://app.local")
	req.AddCookie(&http.Cookie{Name: "safebucket_refresh_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 same-origin, got %d", rr.Code)
	}
}

func TestCSRFGuard_FallsBackToReferer(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/foo", nil)
	req.Header.Set("Referer", "https://example.com/some/page")
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 via Referer, got %d", rr.Code)
	}
}

func TestCSRFGuard_RejectsMissingOriginAndReferer(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when no Origin/Referer, got %d", rr.Code)
	}
}

func TestCSRFGuard_RejectsMFACookieFromForeignOrigin(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://attacker.example")
	req.AddCookie(&http.Cookie{Name: "safebucket_mfa_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for MFA cookie + foreign origin, got %d", rr.Code)
	}
}

func TestCSRFGuard_WildcardAllowsAll(t *testing.T) {
	h := newPassthroughCSRF(t, []string{"*"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/foo", nil)
	req.Header.Set("Origin", "https://anywhere.invalid")
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: "x"})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected wildcard to allow, got %d", rr.Code)
	}
}
