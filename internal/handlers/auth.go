package handlers

import (
	"api/internal/common"
	"context"
	"crypto/rand"
	"encoding/base64"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type OpenIDBeginFunc func(string, string, string) (string, error)
type OpenIDCallbackFunc func(context.Context, string, string, string) (string, string, error)

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

func OpenIDBeginHandler(openidBegin OpenIDBeginFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")

		state, _ := randString(16)
		nonce, _ := randString(16)

		url, err := openidBegin(providerName, state, nonce)
		if err != nil {
			common.RespondWithError(w, http.StatusNotFound, []string{err.Error()})
			return
		}

		setCallbackCookie(w, r, "state", state)
		setCallbackCookie(w, r, "nonce", nonce)

		// Redirect to the OAuth provider URL
		http.Redirect(w, r, url, http.StatusFound)
	}
}

func OpenIDCallbackHandler(openidCallback OpenIDCallbackFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")

		state, err := r.Cookie("state")
		if err != nil {
			common.RespondWithError(w, http.StatusBadRequest, []string{"state not found"})
			return
		}
		if r.URL.Query().Get("state") != state.Value {
			common.RespondWithError(w, http.StatusBadRequest, []string{"state does not match"})
			return
		}

		nonce, err := r.Cookie("nonce")
		if err != nil {
			common.RespondWithError(w, http.StatusInternalServerError, []string{"nonce not found"})
			return
		}

		accessToken, refreshToken, err := openidCallback(
			r.Context(),
			providerName,
			r.URL.Query().Get("code"),
			nonce.Value,
		)

		if err != nil {
			zap.L().Error("Error in OpenIDCallback", zap.Error(err))
			common.RespondWithError(w, http.StatusInternalServerError, []string{err.Error()})
			return
		}

		expiration := time.Now().Add(365 * 24 * time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:    "safebucket_access_token",
			Value:   accessToken,
			Expires: expiration,
			Path:    "/",
		})

		http.SetCookie(w, &http.Cookie{
			Name:    "safebucket_auth_provider",
			Value:   providerName,
			Expires: expiration,
			Path:    "/",
		})

		if refreshToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:    "safebucket_refresh_token",
				Value:   refreshToken,
				Expires: expiration,
				Path:    "/",
			})
		}

		http.Redirect(w, r, "http://localhost:3001/auth/complete", http.StatusFound)
	}
}
