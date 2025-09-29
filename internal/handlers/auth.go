package handlers

import (
	customErr "api/internal/errors"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type OpenIDBeginFunc func(string, string, string) (string, error)
type OpenIDCallbackFunc func(context.Context, *zap.Logger, string, string, string) (string, string, error)

func OpenIDBeginHandler(openidBegin OpenIDBeginFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")

		state, _ := h.RandString(16)
		nonce, _ := h.RandString(16)

		url, err := openidBegin(providerName, state, nonce)
		if err != nil {
			h.RespondWithError(w, http.StatusNotFound, []string{err.Error()})
			return
		}

		h.SetCallbackCookie(w, r, "state", state)
		h.SetCallbackCookie(w, r, "nonce", nonce)

		// Redirect to the OAuth provider URL
		http.Redirect(w, r, url, http.StatusFound)
	}
}

func OpenIDCallbackHandler(webUrl string, openidCallback OpenIDCallbackFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerName := chi.URLParam(r, "provider")

		state, err := r.Cookie("state")
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, []string{"state not found"})
			return
		}
		if r.URL.Query().Get("state") != state.Value {
			h.RespondWithError(w, http.StatusBadRequest, []string{"state does not match"})
			return
		}

		nonce, err := r.Cookie("nonce")
		if err != nil {
			h.RespondWithError(w, http.StatusInternalServerError, []string{"nonce not found"})
			return
		}

		logger := m.GetLogger(r)

		accessToken, refreshToken, err := openidCallback(
			r.Context(),
			logger,
			providerName,
			r.URL.Query().Get("code"),
			nonce.Value,
		)

		if err != nil {
			strErrors := []string{err.Error()}
			var apiErr *customErr.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, strErrors)
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, strErrors)
			}
			return
		}

		expiration := time.Now().Add(365 * 24 * time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:     "safebucket_access_token",
			Value:    accessToken,
			Expires:  expiration,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "safebucket_auth_provider",
			Value:    providerName,
			Expires:  expiration,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})

		if refreshToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "safebucket_refresh_token",
				Value:    refreshToken,
				Expires:  expiration,
				Path:     "/",
				SameSite: http.SameSiteStrictMode,
				Secure:   r.TLS != nil,
			})
		}

		http.Redirect(w, r, fmt.Sprintf("%s/auth/complete", webUrl), http.StatusFound)
	}
}
