package configuration

import (
	"context"
	"fmt"

	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"
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
	LDAPConfig     *ldapclient.Config
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
	idx := 0
	countLocalProviders := 0

	for name, providerCfg := range providersCfg {
		switch providerCfg.Type {
		case models.LocalProviderType:
			if countLocalProviders > 0 {
				zap.L().Fatal("Only one local auth provider can be configured.")
			}
			providers[name] = Provider{
				Name:           string(providerCfg.Type),
				Type:           providerCfg.Type,
				Order:          idx,
				Domains:        providerCfg.Domains,
				SharingOptions: providerCfg.SharingConfiguration,
			}
			countLocalProviders++
			idx++

		case models.OIDCProviderType:
			provider, err := oidc.NewProvider(ctx, providerCfg.OIDC.Issuer)
			if err != nil {
				zap.L().Fatal(
					"Failed to load OIDC provider",
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
				Order:          idx,
				SharingOptions: providerCfg.SharingConfiguration,
			}
			idx++

			zap.L().Info(
				"Loaded auth provider",
				zap.String("name", name),
				zap.String("client_id", providerCfg.OIDC.ClientID),
				zap.String("issuer", providerCfg.OIDC.Issuer),
				zap.Any("domains", providerCfg.Domains),
			)

		case models.LDAPProviderType:
			ldapCfg := &ldapclient.Config{
				URL:          providerCfg.LDAP.URL,
				BindDN:       providerCfg.LDAP.BindDN,
				BindPassword: providerCfg.LDAP.BindPassword,
				BaseDN:       providerCfg.LDAP.BaseDN,
				UserFilter:   providerCfg.LDAP.UserFilter,
				AttributeMap: ldapclient.AttributeMap{
					Email: providerCfg.LDAP.AttributeMap.Email,
				},
				StartTLS:         providerCfg.LDAP.StartTLS,
				TLSInsecureSkip:  providerCfg.LDAP.TLSInsecureSkip,
				ConnectTimeoutMS: providerCfg.LDAP.ConnectTimeoutMS,
			}

			if err := ldapclient.VerifyServiceBind(*ldapCfg); err != nil {
				zap.L().Warn(
					"Failed to verify LDAP service bind at startup; "+
						"the provider is loaded but logins will fail until it is reachable",
					zap.String("name", name),
					zap.String("url", providerCfg.LDAP.URL),
					zap.Error(err),
				)
			}

			providers[name] = Provider{
				Name:           providerCfg.Name,
				Type:           providerCfg.Type,
				Domains:        providerCfg.Domains,
				LDAPConfig:     ldapCfg,
				Order:          idx,
				SharingOptions: providerCfg.SharingConfiguration,
			}
			idx++

			zap.L().Info(
				"Loaded LDAP auth provider",
				zap.String("name", name),
				zap.String("url", providerCfg.LDAP.URL),
				zap.Any("domains", providerCfg.Domains),
			)
		}
	}
	return providers
}
