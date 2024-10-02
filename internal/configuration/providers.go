package configuration

import (
	"api/internal/models"
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"log"
)

type Provider struct {
	Provider    *oidc.Provider
	Verifier    *oidc.IDTokenVerifier
	OauthConfig oauth2.Config
}

type Providers map[string]Provider

type ProvidersConfiguration map[string]models.ProviderConfiguration

func LoadProviders(ctx context.Context, providersCfg ProvidersConfiguration) Providers {
	var providers = Providers{}
	for name, providerCfg := range providersCfg {
		zap.L().Info(
			"auth provider",
			zap.String("name", name),
			zap.String("client_id", providerCfg.ClientId),
			zap.String("issuer", providerCfg.Issuer),
		)

		provider, err := oidc.NewProvider(ctx, providerCfg.Issuer)
		if err != nil {
			log.Fatal(err)
		}

		verifier := provider.Verifier(&oidc.Config{ClientID: providerCfg.ClientId})

		oauthConfig := oauth2.Config{
			ClientID:     providerCfg.ClientId,
			ClientSecret: providerCfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  fmt.Sprintf("http://localhost:1323/auth/providers/%s/callback", name),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		providers[name] = Provider{
			Provider:    provider,
			Verifier:    verifier,
			OauthConfig: oauthConfig,
		}
	}
	return providers
}
