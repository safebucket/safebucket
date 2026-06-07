//go:build integration

package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCPasswordChangeDenied(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			resp, jar := runOIDCAuthCodeFlow(t, app, oidcUserEmail, bootstrap.TestPassword)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, http.StatusFound, resp.StatusCode, "OIDC login should succeed")

			u, _ := url.Parse(app.BaseURL)
			access := cookieFromJar(jar, u, cookieAccessToken)
			require.NotNil(t, access, "access token cookie must be set after OIDC login")

			var user models.User
			require.NoError(t,
				app.DB().Where("email = ?", oidcUserEmail).First(&user).Error,
				"OIDC user must be persisted",
			)
			require.Equal(t, models.OIDCProviderType, user.ProviderType)

			status, codes := app.DoExpectError(t, http.MethodPatch,
				"/api/v1/users/"+user.ID.String(), access.Value,
				models.UserUpdateBody{
					OldPassword: "irrelevant-old-password",
					NewPassword: "new-secure-password",
				})

			assert.Equal(t, http.StatusForbidden, status,
				"an OIDC user must not be allowed to change their password")
			assert.Contains(t, codes, apierrors.CodePasswordChangeNotAllowed,
				"denial must use the dedicated password-change error code")
		})
	}
}
