package helpers

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandString(t *testing.T) {
	t.Run("Valid input", func(t *testing.T) {
		nByte := 16
		result, err := RandString(nByte)
		require.NoError(t, err)
		assert.Len(t, result, 22) // base64 encoding of 16 bytes results in 22 characters

		// Decode the result to ensure it's valid base64
		decoded, err := base64.RawURLEncoding.DecodeString(result)
		require.NoError(t, err)
		assert.Len(t, decoded, nByte)
	})

	t.Run("Zero length input", func(t *testing.T) {
		result, err := RandString(0)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestSetCallbackCookie(t *testing.T) {
	t.Run("Secure connection", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
		r.TLS = &tls.ConnectionState{}

		SetCallbackCookie(w, r, "test_cookie", "test_value")

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "test_cookie", cookies[0].Name)
		assert.Equal(t, "test_value", cookies[0].Value)
		assert.True(t, cookies[0].Secure)
		assert.True(t, cookies[0].HttpOnly)
		assert.Equal(t, int(time.Hour.Seconds()), cookies[0].MaxAge)
	})

	t.Run("Insecure connection", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

		SetCallbackCookie(w, r, "test_cookie", "test_value")

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "test_cookie", cookies[0].Name)
		assert.Equal(t, "test_value", cookies[0].Value)
		assert.False(t, cookies[0].Secure)
		assert.True(t, cookies[0].HttpOnly)
		assert.Equal(t, int(time.Hour.Seconds()), cookies[0].MaxAge)
	})
}
