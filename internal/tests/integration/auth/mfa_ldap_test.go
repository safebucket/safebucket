//go:build integration

package auth_test

import (
	"net/http"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ldapProviderKey = "ldap"
	ldapUserEmail   = "jdoe@example.org"
	ldapUserName    = "jdoe"
)

func ldapUsers() []bootstrap.LDAPUser {
	return []bootstrap.LDAPUser{{
		Email:     ldapUserEmail,
		Password:  bootstrap.TestPassword,
		UID:       ldapUserName,
		FirstName: "John",
		LastName:  "Doe",
	}}
}

func ldapLogin(t *testing.T, app *bootstrap.TestApp) (int, string, string) {
	t.Helper()
	return app.DoLoginCookies(t, http.MethodPost, "/api/v1/auth/providers/"+ldapProviderKey+"/login",
		models.AuthLoginBody{Email: ldapUserEmail, Password: bootstrap.TestPassword})
}

func TestLDAPMFARequiredEnforced(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithLDAP(t, scenario, bootstrap.LDAPSetup{
				ProviderKey: ldapProviderKey,
				MFARequired: true,
				Users:       ldapUsers(),
			})

			koStatus, koAccess, koMFA := app.DoLoginCookies(t, http.MethodPost,
				"/api/v1/auth/providers/"+ldapProviderKey+"/login",
				models.AuthLoginBody{Email: ldapUserEmail, Password: "wrong-password"})
			assert.Equal(t, http.StatusUnauthorized, koStatus, "wrong LDAP password must be rejected")
			assert.Empty(t, koAccess)
			assert.Empty(t, koMFA)

			status, access, mfaToken := ldapLogin(t, app)
			require.Equal(t, http.StatusOK, status)
			assert.Empty(t, access, "no full session before MFA is satisfied")
			require.NotEmpty(t, mfaToken, "restricted MFA token must be issued")

			assertRestrictedTokenDenied(t, app, mfaToken)

			secret, fullAccess := enrollAndVerifyDevice(t, app, mfaToken, "")
			require.NotEmpty(t, fullAccess)

			var user models.User
			require.NoError(t, app.DB().Where("email = ?", ldapUserEmail).First(&user).Error)
			assert.Equal(t, models.LDAPProviderType, user.ProviderType)

			status, access, mfaToken = ldapLogin(t, app)
			require.Equal(t, http.StatusOK, status)
			assert.Empty(t, access, "re-login must still require MFA")
			require.NotEmpty(t, mfaToken)

			status, loginAccess := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: totpAt(t, secret, 30*time.Second)})
			require.Equal(t, http.StatusOK, status, "MFA login verification should succeed")
			require.NotEmpty(t, loginAccess)
		})
	}
}

func TestLDAPMFAOptInEnrollmentEnforced(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithLDAP(t, scenario, bootstrap.LDAPSetup{
				ProviderKey: ldapProviderKey,
				MFARequired: false,
				Users:       ldapUsers(),
			})

			status, access, mfaToken := ldapLogin(t, app)
			require.Equal(t, http.StatusOK, status)
			require.NotEmpty(t, access, "login should succeed without MFA")
			require.Empty(t, mfaToken)

			secret, _ := enrollAndVerifyDevice(t, app, access, bootstrap.TestPassword)
			require.NotEmpty(t, secret)

			status, access, mfaToken = ldapLogin(t, app)
			require.Equal(t, http.StatusOK, status)
			assert.Empty(t, access, "an enrolled device must be enforced at login")
			require.NotEmpty(t, mfaToken)
		})
	}
}

func TestLDAPMFADeviceRemovalRequiresPassword(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithLDAP(t, scenario, bootstrap.LDAPSetup{
				ProviderKey: ldapProviderKey,
				MFARequired: true,
				Users:       ldapUsers(),
			})

			status, _, mfaToken := ldapLogin(t, app)
			require.Equal(t, http.StatusOK, status)
			require.NotEmpty(t, mfaToken)
			_, access := enrollAndVerifyDevice(t, app, mfaToken, "")
			deviceID := firstVerifiedDeviceID(t, app, access)

			status, codes := app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access, models.MFADeviceRemoveBody{})
			assert.Equal(t, http.StatusBadRequest, status, "removal without a password must be rejected")
			assert.Contains(t, codes, apierrors.CodeBadRequest)

			status, codes = app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Password: "wrong-password"})
			assert.Equal(t, http.StatusUnauthorized, status, "removal with a wrong password must be rejected")
			assert.Contains(t, codes, apierrors.CodeInvalidPassword)

			status = app.DoStatus(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Password: bootstrap.TestPassword})
			assert.Equal(t, http.StatusNoContent, status, "removal with the correct password should succeed")
		})
	}
}
