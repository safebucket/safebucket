package configuration

import (
	"context"
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Provider struct {
	Name           string
	Type           models.ProviderType
	Domains        []string
	Provider       *oidc.Provider
	Verifier       *oidc.IDTokenVerifier
	OauthConfig    oauth2.Config
	LDAP           *ldap.Config
	Order          int
	SharingOptions models.SharingConfiguration
}

type Providers map[string]Provider

type ProvidersConfiguration map[string]models.ProviderConfiguration

func LoadProviders(
	ctx context.Context,
	apiURL string,
	providersCfg ProvidersConfiguration,
) Providers {
	providers := Providers{}
	localLoaded := false

	for name, providerCfg := range providersCfg {
		switch providerCfg.Type {
		case models.LocalProviderType:
			if localLoaded {
				zap.L().Fatal("Only one local auth provider can be configured.")
			}
			providers[name] = Provider{
				Name:           string(providerCfg.Type),
				Type:           providerCfg.Type,
				Order:          len(providers),
				Domains:        providerCfg.Domains,
				SharingOptions: providerCfg.SharingConfiguration,
			}
			localLoaded = true

		case models.LDAPProviderType:
			ldapCfg := ldap.Config{
				URL:            providerCfg.LDAP.URL,
				StartTLS:       providerCfg.LDAP.StartTLS,
				SkipTLSVerify:  providerCfg.LDAP.SkipTLSVerify,
				BindDN:         providerCfg.LDAP.BindDN,
				BindPassword:   providerCfg.LDAP.BindPassword,
				SearchBase:     providerCfg.LDAP.SearchBase,
				SearchFilter:   providerCfg.LDAP.SearchFilter,
				EmailAttribute: providerCfg.LDAP.Attributes.Email,
				ConnectTimeout: time.Duration(providerCfg.LDAP.ConnectTimeoutMs) * time.Millisecond,
			}
			if err := ldap.VerifyServiceBind(ldapCfg); err != nil {
				zap.L().Warn(
					"LDAP service-account probe failed; provider will retry on each login",
					zap.String("name", name),
					zap.String("url", ldapCfg.URL),
					zap.Error(err),
				)
			}
			providers[name] = Provider{
				Name:           providerCfg.Name,
				Type:           providerCfg.Type,
				Domains:        providerCfg.Domains,
				LDAP:           &ldapCfg,
				Order:          len(providers),
				SharingOptions: providerCfg.SharingConfiguration,
			}
			zap.L().Info(
				"Loaded auth provider",
				zap.String("name", name),
				zap.String("type", string(providerCfg.Type)),
				zap.String("url", ldapCfg.URL),
				zap.Any("domains", providerCfg.Domains),
			)

		case models.OIDCProviderType:
			provider, err := oidc.NewProvider(ctx, providerCfg.OIDC.Issuer)
			if err != nil {
				zap.L().Fatal(
					"Failed to load provider",
					zap.String("name", name),
					zap.Error(err),
				)
			}
			verifier := provider.Verifier(&oidc.Config{ClientID: providerCfg.OIDC.ClientID})
			oauthConfig := oauth2.Config{
				ClientID:     providerCfg.OIDC.ClientID,
				ClientSecret: providerCfg.OIDC.ClientSecret,
				Endpoint:     provider.Endpoint(),
				RedirectURL:  fmt.Sprintf("%s/api/v1/auth/providers/%s/callback", apiURL, name),
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
			}
			providers[name] = Provider{
				Name:           providerCfg.Name,
				Type:           providerCfg.Type,
				Domains:        providerCfg.Domains,
				Provider:       provider,
				Verifier:       verifier,
				OauthConfig:    oauthConfig,
				Order:          len(providers),
				SharingOptions: providerCfg.SharingConfiguration,
			}
			zap.L().Info(
				"Loaded auth provider",
				zap.String("name", name),
				zap.String("client_id", providerCfg.OIDC.ClientID),
				zap.String("issuer", providerCfg.OIDC.Issuer),
				zap.Any("domains", providerCfg.Domains),
			)
		}
	}
	return providers
}
