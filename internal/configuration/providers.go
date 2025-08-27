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
	Name           string
	Type           string
	Provider       *oidc.Provider
	Verifier       *oidc.IDTokenVerifier
	OauthConfig    oauth2.Config
	Order          int
	SharingOptions models.SharingConfiguration
}

type Providers map[string]Provider

type ProvidersConfiguration map[string]models.ProviderConfiguration

func LoadProviders(ctx context.Context, apiUrl string, providersCfg ProvidersConfiguration) Providers {
	var providers = Providers{}
	idx := 0
	countLocalProviders := 0

	for name, providerCfg := range providersCfg {
		if countLocalProviders == 0 && providerCfg.Type == LocalAuthProviderType {
			providers[name] = Provider{
				Name:           providerCfg.Type,
				Type:           providerCfg.Type,
				Order:          idx,
				SharingOptions: providerCfg.SharingConfiguration,
			}
			countLocalProviders++
			idx++
			continue
		} else if countLocalProviders > 0 && providerCfg.Type == LocalAuthProviderType {
			zap.L().Warn("Only one local auth provider can be configured. Skipping...")
			continue
		}

		provider, err := oidc.NewProvider(ctx, providerCfg.OIDC.Issuer)
		if err != nil {
			zap.L().Fatal(
				"Failed to load provider",
				zap.String("name", name),
				zap.Error(err),
			)
			continue
		}

		verifier := provider.Verifier(&oidc.Config{ClientID: providerCfg.OIDC.ClientId})

		oauthConfig := oauth2.Config{
			ClientID:     providerCfg.OIDC.ClientId,
			ClientSecret: providerCfg.OIDC.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  fmt.Sprintf("%s/api/v1/auth/providers/%s/callback", apiUrl, name),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}

		providers[name] = Provider{
			Name:           providerCfg.Name,
			Type:           providerCfg.Type,
			Provider:       provider,
			Verifier:       verifier,
			OauthConfig:    oauthConfig,
			Order:          idx,
			SharingOptions: providerCfg.SharingConfiguration,
		}

		idx++

		zap.L().Info(
			"Loaded auth provider",
			zap.String("name", name),
			zap.String("client_id", providerCfg.OIDC.ClientId),
			zap.String("issuer", providerCfg.OIDC.Issuer),
		)
	}
	return providers
}
