package services

import (
	c "api/internal/common"
	h "api/internal/helpers"
	"api/internal/models"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/alexedwards/argon2id"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"io"
	"net/http"
	"time"
)

type AuthService struct {
	DB       *gorm.DB
	JWTConf  models.JWTConfiguration
	Config   oauth2.Config
	Verifier *oidc.IDTokenVerifier
	Provider *oidc.Provider
}

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

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(c.Validate[models.AuthLogin]).Post("/login", c.CreateHandler(s.Login))
	// TODO: MFA / ResetPassword / ResetTokens / IsAdmin (:= get user)
	r.With(c.Validate[models.AuthVerify]).Post("/verify", c.CreateHandler(s.Verify))
	r.With(c.Validate[models.AuthVerify]).Post("/refresh", c.CreateHandler(s.Refresh))

	r.Route("/{provider}", func(r chi.Router) {
		r.Get("/begin", s.OAuthBegin)
		r.Get("/callback", s.OAuthCallback)
	})
	return r
}

func (s AuthService) Login(body models.AuthLogin) (models.AuthLoginResponse, error) {
	searchUser := models.User{Email: body.Email, IsExternal: false}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 1 {
		match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
		if err != nil || match == false {
			return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
		}

		accessToken, err := h.NewAccessToken(s.JWTConf.Secret, &searchUser)
		refreshToken, err := h.NewRefreshToken(s.JWTConf.Secret, &searchUser)

		if err != nil {
			return models.AuthLoginResponse{}, errors.New("failed to generate new access token")
		}

		return models.AuthLoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
	}
	return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
}

func (s AuthService) Verify(body models.AuthVerify) (any, error) {
	data, err := h.ParseAccessToken(s.JWTConf.Secret, body.AccessToken)
	return data, err
}

func (s AuthService) Refresh(body models.AuthRefresh) (models.AuthRefreshResponse, error) {
	RefreshToken, err := h.ParseRefreshToken(s.JWTConf.Secret, body.RefreshToken)
	if err != nil {
		return models.AuthRefreshResponse{}, err
	}
	accessToken, err := h.NewAccessToken(s.JWTConf.Secret, &models.User{Email: RefreshToken.Email})
	return models.AuthRefreshResponse{AccessToken: accessToken}, err
}

func (s AuthService) OAuthBegin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	zap.L().Info(
		"Provider",
		zap.String("provider", provider),
	)

	state, _ := randString(16)
	nonce, _ := randString(16)
	setCallbackCookie(w, r, "state", state)
	setCallbackCookie(w, r, "nonce", nonce)

	url := s.Config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s AuthService) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	state, err := r.Cookie("state")
	if err != nil {
		c.RespondWithError(w, http.StatusBadRequest, []string{"state not found"})
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		c.RespondWithError(w, http.StatusBadRequest, []string{"state does not match"})
		return
	}

	oauth2Token, err := s.Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		strErrors := []string{"Failed to exchange token", err.Error()}
		c.RespondWithError(w, http.StatusInternalServerError, strErrors)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.RespondWithError(w, http.StatusInternalServerError, []string{"no id_token field in oauth2 token"})
		return
	}

	idToken, err := s.Verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		c.RespondWithError(w, http.StatusInternalServerError, []string{"failed to verify ID token", err.Error()})
		return
	}

	userInfo, err := s.Provider.UserInfo(r.Context(), oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		c.RespondWithError(w, http.StatusInternalServerError, []string{"failed to get user info", err.Error()})
		return
	}

	nonce, err := r.Cookie("nonce")
	if err != nil {
		c.RespondWithError(w, http.StatusInternalServerError, []string{"nonce not found"})
		return
	}
	if idToken.Nonce != nonce.Value {
		c.RespondWithError(w, http.StatusInternalServerError, []string{"nonce does not match"})
		return
	}

	searchUser := models.User{Email: userInfo.Email, IsExternal: true}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 0 {
		s.DB.Create(&searchUser)
	}

	expiration := time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    "safebucket_access_token",
		Value:   rawIDToken,
		Expires: expiration,
		Path:    "/",
	}
	http.SetCookie(w, &cookie)

	providerCookie := http.Cookie{
		Name:    "safebucket_auth_provider",
		Value:   provider,
		Expires: expiration,
		Path:    "/",
	}
	http.SetCookie(w, &providerCookie)

	if oauth2Token.RefreshToken != "" {
		refreshTokenCookie := http.Cookie{
			Name:    "safebucket_refresh_token",
			Value:   oauth2Token.RefreshToken,
			Expires: expiration,
			Path:    "/",
		}
		http.SetCookie(w, &refreshTokenCookie)
	}
	http.Redirect(w, r, "http://localhost:3001/auth/complete", http.StatusFound)
}
