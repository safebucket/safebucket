//go:build integration

package integration

import "github.com/safebucket/safebucket/internal/models"

func WithLocalSharing(cfg models.Configuration, allowed bool, domains ...string) models.Configuration {
	if cfg.Auth.Providers == nil {
		cfg.Auth.Providers = make(map[string]models.ProviderConfiguration)
	}
	provider := cfg.Auth.Providers["local"]
	provider.SharingConfiguration.Allowed = allowed
	provider.SharingConfiguration.Domains = domains
	cfg.Auth.Providers["local"] = provider
	return cfg
}
