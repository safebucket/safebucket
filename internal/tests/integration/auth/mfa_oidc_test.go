//go:build integration

package auth_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func oidcLogin(t *testing.T, app *bootstrap.TestApp, email string) (int, string, string) {
	t.Helper()

	resp, jar := runOIDCAuthCodeFlow(t, app, email, bootstrap.TestPassword)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusFound, resp.StatusCode, "OIDC callback should redirect")

	u, err := url.Parse(app.BaseURL)
	require.NoError(t, err)

	var access, mfa string
	if c := cookieFromJar(jar, u, cookieAccessToken); c != nil {
		access = c.Value
	}
	if c := cookieFromJar(jar, u, cookieMFAToken); c != nil {
		mfa = c.Value
	}
	return resp.StatusCode, access, mfa
}

func TestOIDCMFARequiredEnforced(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				MFARequired: true,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			_, access, mfaToken := oidcLogin(t, app, oidcUserEmail)
			assert.Empty(t, access, "no access token before MFA is satisfied")
			require.NotEmpty(t, mfaToken, "restricted MFA token must be issued")

			assertRestrictedTokenDenied(t, app, mfaToken)

			secret, fullAccess := enrollAndVerifyDevice(t, app, mfaToken, "")
			require.NotEmpty(t, fullAccess)

			_, access, mfaToken = oidcLogin(t, app, oidcUserEmail)
			assert.Empty(t, access, "second login must still require MFA")
			require.NotEmpty(t, mfaToken)

			status, loginAccess := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: wrongTOTP(t)})
			require.Equal(t, http.StatusUnauthorized, status, "wrong MFA code must be rejected")
			require.Empty(t, loginAccess)

			status, loginAccess = app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: totpAt(t, secret, 30*time.Second)})
			require.Equal(t, http.StatusOK, status, "MFA login verification should succeed")
			require.NotEmpty(t, loginAccess, "full access token issued after MFA")
		})
	}
}

func TestOIDCMFAEnrollmentEnforcedWithoutProviderFlag(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				MFARequired: false,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			_, access, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, access, "login should succeed without MFA")
			require.Empty(t, mfaToken)

			secret, _ := enrollAndVerifyDevice(t, app, access, "")
			require.NotEmpty(t, secret)

			_, access, mfaToken = oidcLogin(t, app, oidcUserEmail)
			assert.Empty(t, access, "an enrolled OIDC device must be enforced at login")
			require.NotEmpty(t, mfaToken, "MFA token expected once a device is enrolled")
		})
	}
}

func TestOIDCMFADeviceRemovalRequiresTOTP(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				MFARequired: true,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)
			secret, access := enrollAndVerifyDevice(t, app, mfaToken, "")
			deviceID := firstVerifiedDeviceID(t, app, access)

			status, codes := app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access, models.MFADeviceRemoveBody{})
			assert.Equal(t, http.StatusBadRequest, status, "removal without a code must be rejected")
			assert.Contains(t, codes, apierrors.CodeBadRequest)

			status, codes = app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Code: wrongTOTP(t)})
			assert.Equal(t, http.StatusUnauthorized, status, "removal with a wrong code must be rejected")
			assert.Contains(t, codes, apierrors.CodeInvalidMFACode)

			status = app.DoStatus(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Code: totpAt(t, secret, 30*time.Second)})
			assert.Equal(t, http.StatusNoContent, status, "removal with a valid code should succeed")
		})
	}
}

func bootOIDCWithEnrolledDevice(t *testing.T, scenario string) (*bootstrap.TestApp, string, string) {
	t.Helper()

	app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
		ProviderKey: oidcProviderKey,
		MFARequired: true,
		Users: []bootstrap.DexUser{
			{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
		},
	})

	_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
	require.NotEmpty(t, mfaToken)
	secret, _ := enrollAndVerifyDevice(t, app, mfaToken, "")
	return app, secret, mfaToken
}

func TestOIDCMFALoginCodeReplayRejected(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app, secret, _ := bootOIDCWithEnrolledDevice(t, scenario)

			_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)

			code := totpAt(t, secret, 30*time.Second)

			status, access := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: code})
			require.Equal(t, http.StatusOK, status, "first verification should succeed")
			require.NotEmpty(t, access)

			status, codes := app.DoExpectError(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: code})
			assert.Equal(t, http.StatusUnauthorized, status, "replayed code must be rejected")
			assert.Contains(t, codes, apierrors.CodeInvalidMFACode)
		})
	}
}

func TestOIDCMFALoginRateLimited(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app, _, _ := bootOIDCWithEnrolledDevice(t, scenario)

			_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)

			for i := 0; i < configuration.MFAMaxAttempts; i++ {
				status, codes := app.DoExpectError(t, http.MethodPost, "/api/v1/auth/mfa/verify",
					mfaToken, models.MFALoginVerifyBody{Code: wrongTOTP(t)})
				assert.Equal(t, http.StatusUnauthorized, status, "attempt %d should be a plain rejection", i+1)
				assert.Contains(t, codes, apierrors.CodeInvalidMFACode)
			}

			status, codes := app.DoExpectError(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: wrongTOTP(t)})
			assert.Equal(t, http.StatusTooManyRequests, status, "attempts beyond the budget must be throttled")
			assert.Contains(t, codes, apierrors.CodeMFARateLimited)
		})
	}
}

func TestOIDCMFASecondDeviceViaRestrictedTokenRejected(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app, _, mfaToken := bootOIDCWithEnrolledDevice(t, scenario)
			status, codes := app.DoExpectError(t, http.MethodPost, "/api/v1/mfa/devices", mfaToken,
				models.MFADeviceSetupBody{Name: "second"})
			assert.Equal(t, http.StatusForbidden, status, "restricted token cannot add a second device")
			assert.Contains(t, codes, apierrors.CodeMFASetupRestricted)
		})
	}
}

func TestOIDCMFADeviceRemovalCodeReplayRejected(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				MFARequired: true,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)
			secret, _ := enrollAndVerifyDevice(t, app, mfaToken, "")

			_, _, mfaToken = oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)
			code := totpAt(t, secret, 30*time.Second)
			status, access := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/mfa/verify",
				mfaToken, models.MFALoginVerifyBody{Code: code})
			require.Equal(t, http.StatusOK, status, "login verification should succeed")
			require.NotEmpty(t, access)

			deviceID := firstVerifiedDeviceID(t, app, access)

			status, codes := app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Code: code})
			assert.Equal(t, http.StatusUnauthorized, status, "a replayed code must not authorize removal")
			assert.Contains(t, codes, apierrors.CodeInvalidMFACode)
		})
	}
}

func TestOIDCMFADeviceRemovalRateLimited(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenarioWithOIDC(t, scenario, bootstrap.OIDCSetup{
				ProviderKey: oidcProviderKey,
				MFARequired: true,
				Users: []bootstrap.DexUser{
					{Email: oidcUserEmail, Password: bootstrap.TestPassword, Username: oidcUserName},
				},
			})

			_, _, mfaToken := oidcLogin(t, app, oidcUserEmail)
			require.NotEmpty(t, mfaToken)
			_, access := enrollAndVerifyDevice(t, app, mfaToken, "")
			deviceID := firstVerifiedDeviceID(t, app, access)

			for i := 0; i < configuration.MFAMaxAttempts; i++ {
				status, codes := app.DoExpectError(t, http.MethodDelete,
					"/api/v1/mfa/devices/"+deviceID.String(), access,
					models.MFADeviceRemoveBody{Code: wrongTOTP(t)})
				assert.Equal(t, http.StatusUnauthorized, status, "attempt %d should be a plain rejection", i+1)
				assert.Contains(t, codes, apierrors.CodeInvalidMFACode)
			}

			status, codes := app.DoExpectError(t, http.MethodDelete,
				"/api/v1/mfa/devices/"+deviceID.String(), access,
				models.MFADeviceRemoveBody{Code: wrongTOTP(t)})
			assert.Equal(t, http.StatusTooManyRequests, status, "attempts beyond the budget must be throttled")
			assert.Contains(t, codes, apierrors.CodeMFARateLimited)
		})
	}
}
