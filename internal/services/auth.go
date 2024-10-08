package services

import (
	"api/internal/configuration"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	"context"
	"errors"
	"fmt"
	"github.com/alexedwards/argon2id"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type AuthService struct {
	DB        *gorm.DB
	JWTConf   models.JWTConfiguration
	Providers configuration.Providers
	WebUrl    string
}

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(h.Validate[models.AuthLogin]).Post("/login", handlers.CreateHandler(s.Login))
	// TODO: MFA / ResetPassword / ResetTokens / IsAdmin (:= get user)
	r.With(h.Validate[models.AuthVerify]).Post("/verify", handlers.CreateHandler(s.Verify))
	r.With(h.Validate[models.AuthVerify]).Post("/refresh", handlers.CreateHandler(s.Refresh))

	r.Route("/providers", func(r chi.Router) {
		r.Get("/", handlers.GetListHandler(s.GetProviderList))
		r.Route("/{provider}", func(r chi.Router) {
			r.Get("/begin", handlers.OpenIDBeginHandler(s.OpenIDBegin))
			r.Get("/callback", handlers.OpenIDCallbackHandler(s.WebUrl, s.OpenIDCallback))
		})
	})
	return r
}

func (s AuthService) Login(body models.AuthLogin) (models.AuthLoginResponse, error) {
	searchUser := models.User{Email: body.Email, IsExternal: false}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 1 {
		match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
		if err != nil || !match {
			return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
		}

		accessToken, err := h.NewAccessToken(s.JWTConf.Secret, &searchUser)
		if err != nil {
			return models.AuthLoginResponse{}, errors.New("failed to generate new access token")
		}

		refreshToken, err := h.NewRefreshToken(s.JWTConf.Secret, &searchUser)
		if err != nil {
			return models.AuthLoginResponse{}, errors.New("failed to generate new refresh token")
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

func (s AuthService) GetProviderList() []models.ProviderResponse {
	var providers = make([]models.ProviderResponse, len(s.Providers))
	for id, provider := range s.Providers {
		providers[provider.Order] = models.ProviderResponse{
			Id:   id,
			Name: provider.Name,
		}
	}
	return providers
}

func (s AuthService) OpenIDBegin(providerName string, state string, nonce string) (string, error) {
	provider, ok := s.Providers[providerName]
	if !ok {
		return "", fmt.Errorf("provider not found")
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, providerName string, code string, nonce string,
) (string, string, error) {
	provider, ok := s.Providers[providerName]
	if !ok {
		return "", "", fmt.Errorf("provider not found")
	}

	oauth2Token, err := provider.OauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange token %s", err.Error())
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", "", fmt.Errorf("no id_token field in oauth2 token")
	}

	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify ID token %s", err.Error())
	}

	if idToken.Nonce != nonce {
		return "", "", fmt.Errorf("nonce does not match")
	}

	userInfo, err := provider.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info %s", err.Error())
	}

	searchUser := models.User{Email: userInfo.Email, IsExternal: true}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 0 {
		s.DB.Create(&searchUser)
	}

	return rawIDToken, oauth2Token.RefreshToken, nil
}
