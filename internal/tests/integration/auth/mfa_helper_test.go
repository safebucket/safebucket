//go:build integration

package auth_test

import (
	"net/http"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/require"
)

const cookieMFAToken = "safebucket_mfa_token"

func totpAt(t *testing.T, secret string, offset time.Duration) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now().Add(offset))
	require.NoError(t, err)
	return code
}

func wrongTOTP(t *testing.T) string {
	t.Helper()
	code, err := totp.GenerateCode("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Now())
	require.NoError(t, err)
	return code
}

func enrollAndVerifyDevice(
	t *testing.T, app *bootstrap.TestApp, token, password string,
) (secret, access string) {
	t.Helper()

	var setup models.MFADeviceSetupResponse
	status := app.Do(t, http.MethodPost, "/api/v1/mfa/devices", token,
		models.MFADeviceSetupBody{Name: "primary", Password: password}, &setup)
	require.Equal(t, http.StatusCreated, status, "device enrollment should succeed")
	require.NotEmpty(t, setup.Secret)
	require.NotEqual(t, uuid.Nil, setup.DeviceID)

	status, access = app.DoGetAuthCookie(t, http.MethodPost,
		"/api/v1/mfa/devices/"+setup.DeviceID.String()+"/verify", token,
		models.MFADeviceVerifyBody{Code: totpAt(t, setup.Secret, 0)})
	require.Equal(t, http.StatusOK, status, "device verification should succeed")
	require.NotEmpty(t, access, "verification should mint a full access token")

	return setup.Secret, access
}

func assertRestrictedTokenDenied(t *testing.T, app *bootstrap.TestApp, token string) {
	t.Helper()
	status, codes := app.DoExpectError(t, http.MethodGet, "/api/v1/buckets", token, nil)
	require.Equal(t, http.StatusForbidden, status, "restricted token must not access protected routes")
	require.Contains(t, codes, apierrors.CodeForbidden)
}

func firstVerifiedDeviceID(t *testing.T, app *bootstrap.TestApp, token string) uuid.UUID {
	t.Helper()
	var list models.MFADevicesListResponse
	status := app.Do(t, http.MethodGet, "/api/v1/mfa/devices", token, nil, &list)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, list.Devices, 1, "expected exactly one verified device")
	return list.Devices[0].ID
}
