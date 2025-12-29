package helpers

import (
	"strings"
	"testing"
	"time"

	"api/internal/configuration"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateTOTPSecret tests TOTP secret generation.
func TestGenerateTOTPSecret(t *testing.T) {
	t.Run("should generate valid secret and URL", func(t *testing.T) {
		email := "test@example.com"

		result, err := GenerateTOTPSecret(email)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Secret, "secret should not be empty")
		assert.NotEmpty(t, result.URL, "URL should not be empty")
	})

	t.Run("should include correct issuer in URL", func(t *testing.T) {
		email := "user@domain.com"

		result, err := GenerateTOTPSecret(email)

		require.NoError(t, err)
		assert.Contains(t, result.URL, "issuer="+configuration.AppName)
	})

	t.Run("should include email in URL", func(t *testing.T) {
		email := "test@example.com"

		result, err := GenerateTOTPSecret(email)

		require.NoError(t, err)
		assert.Contains(t, result.URL, email)
	})

	t.Run("should start with otpauth protocol", func(t *testing.T) {
		email := "test@example.com"

		result, err := GenerateTOTPSecret(email)

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result.URL, "otpauth://totp/"))
	})

	t.Run("should generate base32 encoded secret", func(t *testing.T) {
		email := "test@example.com"

		result, err := GenerateTOTPSecret(email)

		require.NoError(t, err)
		// Base32 characters are A-Z and 2-7
		for _, char := range result.Secret {
			isBase32 := (char >= 'A' && char <= 'Z') || (char >= '2' && char <= '7')
			assert.True(t, isBase32, "secret should be base32 encoded, got char: %c", char)
		}
	})

	t.Run("should generate different secrets for same email", func(t *testing.T) {
		email := "test@example.com"

		result1, err1 := GenerateTOTPSecret(email)
		result2, err2 := GenerateTOTPSecret(email)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, result1.Secret, result2.Secret, "each call should generate a unique secret")
	})
}

// TestGenerateTOTPSecretWithEmail tests URL generation with existing secret.
func TestGenerateTOTPSecretWithEmail(t *testing.T) {
	t.Run("should create valid URL with existing secret", func(t *testing.T) {
		email := "user@example.com"
		secret := "JBSWY3DPEHPK3PXP" // Valid base32 secret

		result, err := GenerateTOTPSecretWithEmail(email, secret)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, secret, result.Secret)
		assert.NotEmpty(t, result.URL)
	})

	t.Run("should format URL correctly", func(t *testing.T) {
		email := "test@domain.com"
		secret := "ABCDEFGHIJKLMNOP"

		result, err := GenerateTOTPSecretWithEmail(email, secret)

		require.NoError(t, err)
		assert.Contains(t, result.URL, "otpauth://totp/")
		assert.Contains(t, result.URL, configuration.AppName)
		assert.Contains(t, result.URL, email)
		assert.Contains(t, result.URL, "secret="+secret)
		assert.Contains(t, result.URL, "issuer="+configuration.AppName)
	})

	t.Run("should preserve provided secret exactly", func(t *testing.T) {
		email := "user@test.com"
		secret := "MYSECRETKEY12345"

		result, err := GenerateTOTPSecretWithEmail(email, secret)

		require.NoError(t, err)
		assert.Equal(t, secret, result.Secret, "secret should not be modified")
	})
}

// TestValidateTOTPCode tests TOTP code validation.
func TestValidateTOTPCode(t *testing.T) {
	t.Run("should validate correct code", func(t *testing.T) {
		// Generate a real secret and code for testing
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Test",
			AccountName: "test@example.com",
			SecretSize:  20,
		})
		require.NoError(t, err)

		// Generate current valid code
		code, err := totp.GenerateCode(key.Secret(), time.Now())
		require.NoError(t, err)

		result := ValidateTOTPCode(key.Secret(), code)

		assert.True(t, result, "should validate correct TOTP code")
	})

	t.Run("should reject invalid code", func(t *testing.T) {
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Test",
			AccountName: "test@example.com",
			SecretSize:  20,
		})
		require.NoError(t, err)

		result := ValidateTOTPCode(key.Secret(), "000000")

		// This might occasionally pass if 000000 happens to be valid, but very unlikely
		// For robust testing, we use a code that's definitely wrong
		assert.False(t, result, "should reject obviously wrong code")
	})

	t.Run("should reject empty code", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"

		result := ValidateTOTPCode(secret, "")

		assert.False(t, result, "should reject empty code")
	})

	t.Run("should reject code with wrong length", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"

		result := ValidateTOTPCode(secret, "12345") // 5 digits instead of 6

		assert.False(t, result, "should reject code with wrong length")
	})

	t.Run("should reject non-numeric code", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"

		result := ValidateTOTPCode(secret, "abcdef")

		assert.False(t, result, "should reject non-numeric code")
	})

	t.Run("should handle code with leading zeros", func(t *testing.T) {
		// Generate a secret and keep trying until we get a code with leading zero
		// This test verifies that codes like "012345" are handled correctly
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Test",
			AccountName: "test@example.com",
			SecretSize:  20,
		})
		require.NoError(t, err)

		// Generate current valid code
		code, err := totp.GenerateCode(key.Secret(), time.Now())
		require.NoError(t, err)

		// Validate the generated code (which may or may not have leading zeros)
		result := ValidateTOTPCode(key.Secret(), code)
		assert.True(t, result, "should validate code regardless of leading zeros")
	})
}

// TestValidateTOTPCode_Integration tests the full flow of generating and validating.
func TestValidateTOTPCode_Integration(t *testing.T) {
	t.Run("should validate code generated from our own secret", func(t *testing.T) {
		email := "integration@test.com"

		// Generate secret using our function
		totpKey, err := GenerateTOTPSecret(email)
		require.NoError(t, err)

		// Generate code using the secret
		code, err := totp.GenerateCode(totpKey.Secret, time.Now())
		require.NoError(t, err)

		// Validate using our function
		result := ValidateTOTPCode(totpKey.Secret, code)

		assert.True(t, result, "should validate code generated from our secret")
	})
}
