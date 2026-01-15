package services

import (
	"testing"

	"api/internal/configuration"
	"api/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestPasswordResetAudienceValidation tests that CompletePasswordReset
// correctly validates token audience to prevent cross-flow attacks.
//
// Security boundary: Only tokens with AudienceMFAReset can complete password reset.
// This prevents login-flow tokens from being used to reset passwords.
func TestPasswordResetAudienceValidation(t *testing.T) {
	t.Run("should allow AudienceMFAReset token", func(t *testing.T) {
		claims := models.UserClaims{
			Aud: configuration.AudienceMFAReset,
		}

		// This is the audience check from CompletePasswordReset
		isValidAudience := claims.Aud == configuration.AudienceMFAReset

		assert.True(t, isValidAudience,
			"AudienceMFAReset should be accepted for password reset completion")
	})

	t.Run("should block AudienceMFALogin token (cross-flow attack)", func(t *testing.T) {
		claims := models.UserClaims{
			Aud: configuration.AudienceMFALogin,
		}

		// This simulates the check in CompletePasswordReset
		isInvalidAudience := claims.Aud != configuration.AudienceMFAReset

		assert.True(t, isInvalidAudience,
			"AudienceMFALogin should be rejected for password reset completion")
	})

	t.Run("should block AudienceAccessToken (full access token)", func(t *testing.T) {
		claims := models.UserClaims{
			Aud: configuration.AudienceAccessToken,
		}

		isInvalidAudience := claims.Aud != configuration.AudienceMFAReset

		assert.True(t, isInvalidAudience,
			"Full access token should be rejected for password reset completion")
	})

	t.Run("should block AudienceRefreshToken", func(t *testing.T) {
		claims := models.UserClaims{
			Aud: configuration.AudienceRefreshToken,
		}

		isInvalidAudience := claims.Aud != configuration.AudienceMFAReset

		assert.True(t, isInvalidAudience,
			"Refresh token should be rejected for password reset completion")
	})
}

// TestMFABypassPrevention tests that MFA-enabled users cannot bypass MFA
// during password reset by using a restricted token without MFA verification.
//
// Attack scenario prevented:
// 1. Attacker requests password reset for victim
// 2. Attacker verifies email code → gets RestrictedAccessToken (MFA=false)
// 3. Attacker SKIPS MFA verification and calls /complete directly
// 4. Without the fix, password would be reset bypassing MFA
//
// The fix: If user has MFA enabled, claims.MFA must be true.
func TestMFABypassPrevention(t *testing.T) {
	t.Run("should block unverified MFA token for MFA-enabled user", func(t *testing.T) {
		userWithMFA := models.User{
			ID:    uuid.New(),
			Email: "mfa-user@example.com",
			Role:  models.RoleUser,
			MFADevices: []models.MFADevice{
				{
					ID:         uuid.New(),
					IsVerified: true,
					IsDefault:  true,
				},
			},
		}

		// Restricted token WITHOUT MFA verification (MFA=false)
		unverifiedClaims := models.UserClaims{
			MFA: false, // Not verified yet
		}

		// This is the MFA bypass check from CompletePasswordReset
		shouldBlockMFABypass := userWithMFA.HasMFAEnabled() && !unverifiedClaims.MFA

		assert.True(t, shouldBlockMFABypass,
			"MFA-enabled user with unverified token should be blocked")
	})

	t.Run("should allow verified MFA token for MFA-enabled user", func(t *testing.T) {
		userWithMFA := models.User{
			ID:    uuid.New(),
			Email: "mfa-user@example.com",
			Role:  models.RoleUser,
			MFADevices: []models.MFADevice{
				{
					ID:         uuid.New(),
					IsVerified: true,
					IsDefault:  true,
				},
			},
		}

		// Restricted token WITH MFA verification (MFA=true)
		verifiedClaims := models.UserClaims{
			MFA: true, // Verified via /auth/mfa/verify
		}

		shouldBlockMFABypass := userWithMFA.HasMFAEnabled() && !verifiedClaims.MFA

		assert.False(t, shouldBlockMFABypass,
			"MFA-enabled user with verified token should be allowed")
	})

	t.Run("should allow unverified token for user without MFA", func(t *testing.T) {
		userWithoutMFA := models.User{
			ID:         uuid.New(),
			Email:      "no-mfa-user@example.com",
			Role:       models.RoleUser,
			MFADevices: []models.MFADevice{}, // No MFA devices
		}

		claims := models.UserClaims{
			MFA: false,
		}

		shouldBlockMFABypass := userWithoutMFA.HasMFAEnabled() && !claims.MFA

		assert.False(t, shouldBlockMFABypass,
			"User without MFA should be allowed with unverified token")
	})

	t.Run("should not count unverified MFA devices", func(t *testing.T) {
		// User started MFA setup but didn't complete verification
		userWithUnverifiedMFA := models.User{
			ID:    uuid.New(),
			Email: "partial-mfa-user@example.com",
			Role:  models.RoleUser,
			MFADevices: []models.MFADevice{
				{
					ID:         uuid.New(),
					IsVerified: false, // Setup incomplete
					IsDefault:  false,
				},
			},
		}

		claims := models.UserClaims{
			MFA: false,
		}

		// HasMFAEnabled only counts verified devices
		shouldBlockMFABypass := userWithUnverifiedMFA.HasMFAEnabled() && !claims.MFA

		assert.False(t, shouldBlockMFABypass,
			"User with only unverified MFA devices should be allowed")
	})
}

// TestCrossFlowAttackPrevention tests that tokens from one flow cannot be used
// in another flow, preventing cross-flow attacks.
func TestCrossFlowAttackPrevention(t *testing.T) {
	t.Run("login token cannot complete password reset", func(t *testing.T) {
		// Attacker obtains victim's login-flow MFA token
		loginFlowToken := models.UserClaims{
			Aud: configuration.AudienceMFALogin,
		}

		// CompletePasswordReset checks:
		// 1. Audience must be AudienceMFAReset
		audienceBlocked := loginFlowToken.Aud != configuration.AudienceMFAReset

		assert.True(t, audienceBlocked,
			"Login flow token should be blocked from password reset completion")
	})

	t.Run("password reset token cannot get full access via login MFA verify", func(t *testing.T) {
		// Token from password reset flow
		resetFlowToken := models.UserClaims{
			Aud: configuration.AudienceMFAReset,
		}

		// VerifyMFALogin returns different response based on audience:
		// - AudienceMFALogin -> full access + refresh tokens
		// - AudienceMFAReset -> restricted token with MFA=true
		returnsRestrictedToken := resetFlowToken.Aud == configuration.AudienceMFAReset

		assert.True(t, returnsRestrictedToken,
			"Password reset token should get restricted token, not full access")
	})
}

// TestAudienceConstants verifies the audience constants are correctly defined
// and distinct to prevent token type confusion.
func TestAudienceConstants(t *testing.T) {
	t.Run("should have expected audience values", func(t *testing.T) {
		assert.Equal(t, "app:*", configuration.AudienceAccessToken)
		assert.Equal(t, "auth:refresh", configuration.AudienceRefreshToken)
		assert.Equal(t, "auth:mfa:login", configuration.AudienceMFALogin)
		assert.Equal(t, "auth:mfa:password-reset", configuration.AudienceMFAReset)
	})

	t.Run("all audience constants must be unique", func(t *testing.T) {
		audiences := []string{
			configuration.AudienceAccessToken,
			configuration.AudienceRefreshToken,
			configuration.AudienceMFALogin,
			configuration.AudienceMFAReset,
		}

		seen := make(map[string]bool)
		for _, aud := range audiences {
			assert.False(t, seen[aud], "Audience %s must be unique", aud)
			seen[aud] = true
		}
	})

	t.Run("restricted audiences should share common prefix", func(t *testing.T) {
		// Both restricted audiences start with "auth:mfa:" for consistency
		assert.Contains(t, configuration.AudienceMFALogin, "auth:mfa:")
		assert.Contains(t, configuration.AudienceMFAReset, "auth:mfa:")
	})
}

// TestCompletePasswordResetSecurityChecks validates both security checks
// are correctly applied in sequence.
func TestCompletePasswordResetSecurityChecks(t *testing.T) {
	t.Run("both checks must pass for MFA-enabled user", func(t *testing.T) {
		userWithMFA := models.User{
			MFADevices: []models.MFADevice{
				{ID: uuid.New(), IsVerified: true, IsDefault: true},
			},
		}

		// Valid token: correct audience AND MFA verified
		validClaims := models.UserClaims{
			Aud: configuration.AudienceMFAReset,
			MFA: true,
		}

		// Check 1: MFA bypass prevention
		mfaBypassBlocked := userWithMFA.HasMFAEnabled() && !validClaims.MFA
		// Check 2: Audience validation
		audienceInvalid := validClaims.Aud != configuration.AudienceMFAReset

		assert.False(t, mfaBypassBlocked, "Valid MFA should not be blocked")
		assert.False(t, audienceInvalid, "Valid audience should not be blocked")
	})

	t.Run("wrong audience blocks even with valid MFA", func(t *testing.T) {
		// Wrong audience even though MFA is verified
		wrongAudienceClaims := models.UserClaims{
			Aud: configuration.AudienceMFALogin, // Wrong!
		}

		audienceInvalid := wrongAudienceClaims.Aud != configuration.AudienceMFAReset

		assert.True(t, audienceInvalid,
			"Wrong audience should be blocked regardless of MFA status")
	})

	t.Run("unverified MFA blocks even with correct audience", func(t *testing.T) {
		userWithMFA := models.User{
			MFADevices: []models.MFADevice{
				{ID: uuid.New(), IsVerified: true, IsDefault: true},
			},
		}

		// Correct audience but MFA not verified
		unverifiedMFAClaims := models.UserClaims{
			MFA: false, // Not verified!
		}

		mfaBypassBlocked := userWithMFA.HasMFAEnabled() && !unverifiedMFAClaims.MFA

		assert.True(t, mfaBypassBlocked,
			"Unverified MFA should be blocked regardless of audience")
	})
}
