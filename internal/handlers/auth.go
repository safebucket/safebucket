package handlers

import (
	"context"
	"fmt"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type (
	OpenIDBeginFunc    func(string, string, string) (string, error)
	OpenIDCallbackFunc func(context.Context, *zap.Logger, string, string, string) (models.OIDCCallbackResult, error)
)

func providerKeyFromURL(r *http.Request) (string, error) {
	key := chi.URLParam(r, "provider")
	if err := h.ValidateProviderName(key); err != nil {
		return "", apierrors.New(http.StatusBadRequest, apierrors.CodeInvalidProviderName)
	}
	return key, nil
}

func OpenIDBeginHandler(openidBegin OpenIDBeginFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), "handlers.OpenIDBegin")
		defer span.End()
		r = r.WithContext(ctx)

		providerName, err := providerKeyFromURL(r)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		state, _ := h.RandString(16)
		nonce, _ := h.RandString(16)

		url, err := openidBegin(providerName, state, nonce)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		h.SetCallbackCookie(w, r, "state", state)
		h.SetCallbackCookie(w, r, "nonce", nonce)

		http.Redirect(w, r, url, http.StatusFound)
	}
}

func OpenIDCallbackHandler(webURL string, cookieSecureForce bool, openidCallback OpenIDCallbackFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), "handlers.OpenIDCallback")
		defer span.End()
		r = r.WithContext(ctx)

		providerName, err := providerKeyFromURL(r)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		state, err := r.Cookie("state")
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeOIDCStateNotFound})
			return
		}
		if r.URL.Query().Get("state") != state.Value {
			h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeOIDCStateMismatch})
			return
		}

		nonce, err := r.Cookie("nonce")
		if err != nil {
			h.RespondWithError(w, http.StatusBadRequest, []string{apierrors.CodeOIDCNonceNotFound})
			return
		}

		logger := m.GetLogger(r)

		result, err := openidCallback(
			r.Context(),
			logger,
			providerName,
			r.URL.Query().Get("code"),
			nonce.Value,
		)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		if result.MFARequired {
			SetMFACookie(w, r, result.MFAToken, cookieSecureForce)
			http.Redirect(w, r, fmt.Sprintf("%s/auth/complete?mfa=required", webURL), http.StatusFound)
			return
		}

		SetAuthCookies(w, r, result.AccessToken, result.RefreshToken, providerName, cookieSecureForce)

		http.Redirect(w, r, fmt.Sprintf("%s/auth/complete", webURL), http.StatusFound)
	}
}
