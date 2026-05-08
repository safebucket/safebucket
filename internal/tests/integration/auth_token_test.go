//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const probeEndpoint = "/api/v1/buckets"

func TestAuthTokenValidation(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			user := app.CreateUser(t, "tokenvalidation@example.com")
			validToken := app.LoginAs(t, user.Email)

			t.Run("valid token is accepted", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, validToken, nil)
				assert.Equal(t, http.StatusOK, status)
			})

			t.Run("wrong JWT secret is rejected", func(t *testing.T) {
				wrongToken := craftTokenWithWrongSecret(t, user.ID, user.Email, user.Role)
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, wrongToken, nil)
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("tampered payload is rejected", func(t *testing.T) {
				tampered := tamperedPayloadToken(t, validToken)
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, tampered, nil)
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("expired token is rejected", func(t *testing.T) {
				expired := craftExpiredToken(t, app.Config.App.JWTSecret, user.ID, user.Email, user.Role)
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, expired, nil)
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("missing Authorization header is rejected", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, "", nil)
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("malformed JWT format is rejected", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, probeEndpoint, "not.a.valid.jwt", nil)
				assert.Equal(t, http.StatusForbidden, status)
			})
		})
	}
}

func TestAuthPasswordReset(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			t.Run("full reset flow succeeds", func(t *testing.T) {
				user := app.CreateUser(t, "resetfull@example.com")

				code, challengeID := requestReset(t, app, user.Email)

				status, restrictedToken := app.doGetMFACookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/validate", challengeID),
					"", models.PasswordResetValidateBody{Code: code})
				require.Equal(t, http.StatusCreated, status)
				require.NotEmpty(t, restrictedToken, "MFA cookie should be set after reset validate")

				const newPassword = "correcthorsebatterystaple2"
				status, _ = app.doGetAuthCookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeID),
					restrictedToken, models.PasswordResetCompleteBody{NewPassword: newPassword})
				require.Equal(t, http.StatusNoContent, status)

				status = app.DoStatus(t, http.MethodPost, "/api/v1/auth/login", "",
					models.AuthLoginBody{Email: user.Email, Password: newPassword})
				require.Equal(t, http.StatusOK, status)
			})

			t.Run("cross-user restricted token is rejected on complete", func(t *testing.T) {
				userA := app.CreateUser(t, "reseta@example.com")
				userB := app.CreateUser(t, "resetb@example.com")

				codeA, challengeIDA := requestReset(t, app, userA.Email)

				status, tokenA := app.doGetMFACookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/validate", challengeIDA),
					"", models.PasswordResetValidateBody{Code: codeA})
				require.Equal(t, http.StatusCreated, status)
				require.NotEmpty(t, tokenA)

				_, challengeIDB := requestReset(t, app, userB.Email)

				status = app.DoStatus(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeIDB),
					tokenA, models.PasswordResetCompleteBody{NewPassword: "newpassword123"})
				assert.Equal(t, http.StatusBadRequest, status)
			})

			t.Run("wrong-audience token is rejected on complete", func(t *testing.T) {
				user := app.CreateUser(t, "resetwrongaud@example.com")

				code, challengeID := requestReset(t, app, user.Email)

				status, _ := app.doGetMFACookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/validate", challengeID),
					"", models.PasswordResetValidateBody{Code: code})
				require.Equal(t, http.StatusCreated, status)

				challengeUUID := uuid.MustParse(challengeID)
				wrongAudToken := craftTokenWithAudience(t, app.Config.App.JWTSecret,
					user.ID, user.Email, user.Role, configuration.AudienceMFALogin, &challengeUUID)

				status = app.DoStatus(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeID),
					wrongAudToken, models.PasswordResetCompleteBody{NewPassword: "newpassword123"})
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("password reset revokes pre-existing sessions", func(t *testing.T) {
				user := app.CreateUser(t, "resetrevoke@example.com")
				oldToken := app.LoginAs(t, user.Email)

				require.Equal(t, http.StatusOK,
					app.DoStatus(t, http.MethodGet, probeEndpoint, oldToken, nil),
					"old token should work before reset")

				code, challengeID := requestReset(t, app, user.Email)

				status, restrictedToken := app.doGetMFACookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/validate", challengeID),
					"", models.PasswordResetValidateBody{Code: code})
				require.Equal(t, http.StatusCreated, status)
				require.NotEmpty(t, restrictedToken)

				completeStatus, newToken := app.doGetAuthCookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeID),
					restrictedToken,
					models.PasswordResetCompleteBody{NewPassword: "newpassword456"})
				require.Equal(t, http.StatusNoContent, completeStatus)

				assert.Equal(t, http.StatusUnauthorized,
					app.DoStatus(t, http.MethodGet, probeEndpoint, oldToken, nil),
					"old token must be rejected as session-revoked after password reset")
				assert.Equal(t, http.StatusOK,
					app.DoStatus(t, http.MethodGet, probeEndpoint, newToken, nil),
					"new token issued by reset must work")
			})

			t.Run("completed challenge cannot be reused", func(t *testing.T) {
				user := app.CreateUser(t, "resetreuse@example.com")

				code, challengeID := requestReset(t, app, user.Email)

				status, restrictedToken := app.doGetMFACookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/validate", challengeID),
					"", models.PasswordResetValidateBody{Code: code})
				require.Equal(t, http.StatusCreated, status)
				require.NotEmpty(t, restrictedToken)

				firstStatus, _ := app.doGetAuthCookie(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeID),
					restrictedToken, models.PasswordResetCompleteBody{NewPassword: "firstpassword123"})
				require.Equal(t, http.StatusNoContent, firstStatus)

				status = app.DoStatus(t, http.MethodPost,
					fmt.Sprintf("/api/v1/auth/reset-password/%s/complete", challengeID),
					restrictedToken, models.PasswordResetCompleteBody{NewPassword: "secondpassword123"})
				assert.Equal(t, http.StatusBadRequest, status)
			})
		})
	}
}

func requestReset(t *testing.T, app *TestApp, email string) (code, challengeID string) {
	t.Helper()

	status := app.DoStatus(t, http.MethodPost, "/api/v1/auth/reset-password", "",
		models.PasswordResetRequestBody{Email: email})
	require.Equal(t, http.StatusCreated, status)

	app.Eventually(t, func() bool {
		for _, n := range app.ReadNotifications(t) {
			if n.To != email || n.TemplateName != "password_reset" {
				continue
			}
			var payload struct {
				Secret       string
				ChallengeURL string
			}
			if err := json.Unmarshal(n.Args, &payload); err != nil || payload.Secret == "" {
				continue
			}
			parts := strings.Split(strings.TrimRight(payload.ChallengeURL, "/"), "/")
			if len(parts) == 0 {
				continue
			}
			code = payload.Secret
			challengeID = parts[len(parts)-1]
			return true
		}
		return false
	}, "password_reset notification for "+email)

	require.NotEmpty(t, code, "notification must contain reset code")
	require.NotEmpty(t, challengeID, "notification URL must contain challenge ID")
	return code, challengeID
}

func craftTokenWithWrongSecret(t *testing.T, userID uuid.UUID, email string, role models.Role) string {
	t.Helper()
	claims := models.UserClaims{
		Email:    email,
		UserID:   userID,
		Role:     role,
		Provider: string(models.LocalProviderType),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{configuration.AudienceAccessToken},
			Issuer:    configuration.AppName,
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(1 * time.Hour)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("wrong-secret-key"))
	require.NoError(t, err)
	return signed
}

func craftExpiredToken(t *testing.T, jwtSecret string, userID uuid.UUID, email string, role models.Role) string {
	t.Helper()
	claims := models.UserClaims{
		Email:    email,
		UserID:   userID,
		Role:     role,
		Provider: string(models.LocalProviderType),
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{configuration.AudienceAccessToken},
			Issuer:    configuration.AppName,
			IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-2 * time.Hour)},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-1 * time.Hour)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return signed
}

func craftTokenWithAudience(
	t *testing.T,
	jwtSecret string,
	userID uuid.UUID,
	email string,
	role models.Role,
	audience string,
	challengeID *uuid.UUID,
) string {
	t.Helper()
	claims := models.UserClaims{
		Email:       email,
		UserID:      userID,
		Role:        role,
		Provider:    string(models.LocalProviderType),
		ChallengeID: challengeID,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audience},
			Issuer:    configuration.AppName,
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(1 * time.Hour)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return signed
}

func tamperedPayloadToken(t *testing.T, jweToken string) string {
	t.Helper()
	parts := strings.Split(jweToken, ".")
	require.Len(t, parts, 5, "expected 5-part JWE")
	// Corrupt the ciphertext — GCM authentication tag verification will fail.
	cs := []byte(parts[3])
	cs[0] ^= 0x01
	parts[3] = string(cs)
	return strings.Join(parts, ".")
}
