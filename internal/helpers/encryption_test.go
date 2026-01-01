package helpers

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptSecret tests AES-256-GCM encryption.
func TestEncryptSecret(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012") // 32 bytes

	t.Run("should encrypt and return base64 encoded string", func(t *testing.T) {
		plaintext := "test-secret-value"

		result, err := EncryptSecret(plaintext, validKey)

		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// Verify it's valid base64
		decoded, err := base64.StdEncoding.DecodeString(result)
		require.NoError(t, err)
		assert.Greater(t, len(decoded), len(plaintext), "ciphertext should be longer due to nonce and auth tag")
	})

	t.Run("should produce different ciphertext for same plaintext", func(t *testing.T) {
		plaintext := "same-secret"

		encrypted1, err1 := EncryptSecret(plaintext, validKey)
		encrypted2, err2 := EncryptSecret(plaintext, validKey)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, encrypted1, encrypted2, "different random nonces should produce different ciphertexts")
	})

	t.Run("should reject key size less than 32 bytes", func(t *testing.T) {
		plaintext := "test"
		shortKey := []byte("short-key") // 9 bytes

		_, err := EncryptSecret(plaintext, shortKey)

		assert.Error(t, err)
		assert.Equal(t, "encryption key must be 32 bytes for AES-256", err.Error())
	})

	t.Run("should reject key size greater than 32 bytes", func(t *testing.T) {
		plaintext := "test"
		longKey := []byte("1234567890123456789012345678901234567890") // 40 bytes

		_, err := EncryptSecret(plaintext, longKey)

		assert.Error(t, err)
		assert.Equal(t, "encryption key must be 32 bytes for AES-256", err.Error())
	})

	t.Run("should handle empty plaintext", func(t *testing.T) {
		plaintext := ""

		result, err := EncryptSecret(plaintext, validKey)

		require.NoError(t, err)
		assert.NotEmpty(t, result, "should still produce ciphertext for empty string")
	})
}

// TestDecryptSecret tests AES-256-GCM decryption.
func TestDecryptSecret(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012") // 32 bytes

	t.Run("should decrypt valid ciphertext", func(t *testing.T) {
		original := "secret-to-decrypt"

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)

		require.NoError(t, err)
		assert.Equal(t, original, decrypted)
	})

	t.Run("should reject key size not equal to 32 bytes", func(t *testing.T) {
		encrypted := "some-base64-value"
		wrongSizeKey := []byte("too-short")

		_, err := DecryptSecret(encrypted, wrongSizeKey)

		assert.Error(t, err)
		assert.Equal(t, "encryption key must be 32 bytes for AES-256", err.Error())
	})

	t.Run("should reject invalid base64 input", func(t *testing.T) {
		invalidBase64 := "not-valid-base64!!!"

		_, err := DecryptSecret(invalidBase64, validKey)

		assert.Error(t, err)
	})

	t.Run("should reject ciphertext too short", func(t *testing.T) {
		// GCM nonce is 12 bytes, so anything shorter should fail
		shortCiphertext := base64.StdEncoding.EncodeToString([]byte("short"))

		_, err := DecryptSecret(shortCiphertext, validKey)

		assert.Error(t, err)
		assert.Equal(t, "ciphertext too short", err.Error())
	})

	t.Run("should fail with wrong key", func(t *testing.T) {
		original := "secret-value"
		wrongKey := []byte("abcdefghijklmnopqrstuvwxyz123456") // Different 32-byte key

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		_, err = DecryptSecret(encrypted, wrongKey)

		assert.Error(t, err, "decryption with wrong key should fail")
	})

	t.Run("should fail with tampered ciphertext", func(t *testing.T) {
		original := "secret-value"

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		// Tamper with the ciphertext
		decoded, _ := base64.StdEncoding.DecodeString(encrypted)
		if len(decoded) > 0 {
			decoded[len(decoded)-1] ^= 0xFF // Flip bits in last byte
		}
		tampered := base64.StdEncoding.EncodeToString(decoded)

		_, err = DecryptSecret(tampered, validKey)

		assert.Error(t, err, "tampered ciphertext should fail authentication")
	})
}

// TestEncryptDecrypt_RoundTrip tests encryption followed by decryption.
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	validKey := []byte("12345678901234567890123456789012") // 32 bytes

	t.Run("should encrypt then decrypt to original plaintext", func(t *testing.T) {
		original := "my-secret-password-123"

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		original := ""

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})

	t.Run("should handle unicode characters", func(t *testing.T) {
		original := "密码 пароль 🔐 مرمز"

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})

	t.Run("should handle long strings", func(t *testing.T) {
		// Generate a long string (1000 characters)
		original := strings.Repeat("a", 1000)

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})

	t.Run("should handle special characters", func(t *testing.T) {
		original := `!@#$%^&*()_+-=[]{}|;':",.<>?/\` + "`"

		encrypted, err := EncryptSecret(original, validKey)
		require.NoError(t, err)

		decrypted, err := DecryptSecret(encrypted, validKey)
		require.NoError(t, err)

		assert.Equal(t, original, decrypted)
	})
}
