//go:build integration

package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	oidcUserEmail = "alice@example.com"
	oidcUserName  = "alice"
)

func TestOIDC(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				Users:       []bootstrap.DexUser{{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName}},
			})

			resp, jar := runOIDCAuthCodeFlow(t, app, oidcUserEmail, bootstrap.TestPassword)
			defer resp.Body.Close()

			require.Equal(t, http.StatusFound, resp.StatusCode, "callback should 302 to webURL on success")

			loc, err := resp.Location()
			require.NoError(t, err)
			assert.Equal(t, app.Config.App.WebURL+"/auth/complete", loc.String())

			u, _ := url.Parse(app.BaseURL)
			access := cookieFromJar(jar, u, cookieAccessToken)
			require.NotNil(t, access, "access token cookie must be set on success")
			require.NotEmpty(t, access.Value)
			refresh := cookieFromJar(jar, u, cookieRefreshToken)
			require.NotNil(t, refresh, "refresh token cookie must be set on success")

			var user models.User
			require.NoError(t,
				app.DB().Where("email = ?", oidcUserEmail).First(&user).Error,
				"user must be persisted in DB",
			)
			assert.Equal(t, oidcUserEmail, user.Email)
			assert.Equal(t, models.OIDCProviderType, user.ProviderType)
			assert.Equal(t, oidcProviderKey, user.ProviderKey)
			assert.Equal(t, models.RoleUser, user.Role)
		})
	}
}
