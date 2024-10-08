package configuration

import (
	"api/internal/models"
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Provider struct {
	Name        string
	Provider    *oidc.Provider
	Verifier    *oidc.IDTokenVerifier
	OauthConfig oauth2.Config
	Order       int
}

type Providers map[string]Provider

type ProvidersConfiguration map[string]models.ProviderConfiguration

func LoadProviders(ctx context.Context, apiUrl string, providersCfg ProvidersConfiguration) Providers {
	var providers = Providers{}
	idx := 0

	for name, providerCfg := range providersCfg {
		provider, err := oidc.NewProvider(ctx, providerCfg.Issuer)
		if err != nil {
			zap.L().Error(
				"Failed to load provider",
				zap.String("name", name),
				zap.Error(err),
			)
			continue
		}

		verifier := provider.Verifier(&oidc.Config{ClientID: providerCfg.ClientId})

		oauthConfig := oauth2.Config{
			ClientID:     providerCfg.ClientId,
			ClientSecret: providerCfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  fmt.Sprintf("%s/auth/providers/%s/callback", apiUrl, name),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		providers[name] = Provider{
			Name:        providerCfg.Name,
			Provider:    provider,
			Verifier:    verifier,
			OauthConfig: oauthConfig,
			Order:       idx,
		}

		idx++

		zap.L().Info(
			"Loaded auth provider",
			zap.String("name", name),
			zap.String("client_id", providerCfg.ClientId),
			zap.String("issuer", providerCfg.Issuer),
		)
	}
	return providers
}
