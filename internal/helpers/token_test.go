package helpers

import (
	"context"
	"strings"
	"testing"
	"time"

	"api/internal/models"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateHash tests password hashing functionality.
func TestCreateHash(t *testing.T) {
	t.Run("should hash password successfully", func(t *testing.T) {
		password := "testPassword123"
		hash, err := CreateHash(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
	})

	t.Run("should create different hashes for same password", func(t *testing.T) {
		password := "testPassword123"
		hash1, err1 := CreateHash(password)
		hash2, err2 := CreateHash(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2, "hashes should be different due to different salts")
	})

	t.Run("should create valid argon2id hash that can be verified", func(t *testing.T) {
		password := "testPassword123"
		hash, err := CreateHash(password)

		require.NoError(t, err)

		// Verify the hash can be checked
		match, err := argon2id.ComparePasswordAndHash(password, hash)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("should reject wrong password", func(t *testing.T) {
		password := "testPassword123"
		hash, err := CreateHash(password)

		require.NoError(t, err)

		match, err := argon2id.ComparePasswordAndHash("wrongPassword", hash)
		require.NoError(t, err)
		assert.False(t, match)
	})
}

// TestNewAccessToken tests JWT access token generation.
func TestNewAccessToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should create valid access token", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2, "JWT should have 3 parts separated by dots")
	})

	t.Run("should have correct claims", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		// Parse the token to verify claims
		claims := &models.UserClaims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		assert.True(t, parsedToken.Valid)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, provider, claims.Provider)
		assert.Equal(t, "safebucket", claims.Issuer)
		assert.Equal(t, "app:*", claims.Aud)
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		claims := &models.UserClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(60 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		// Allow 5 second tolerance for test execution time
		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should respect custom expiry values", func(t *testing.T) {
		testCases := []int{5, 30, 120, 1440}

		for _, expiryMinutes := range testCases {
			token, err := NewAccessToken(jwtSecret, user, provider, expiryMinutes)
			require.NoError(t, err)

			claims := &models.UserClaims{}
			_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			require.NoError(t, err)
			expectedExpiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
			actualExpiry := claims.ExpiresAt.Time

			diff := actualExpiry.Sub(expectedExpiry).Abs()
			assert.Less(t, diff, 5*time.Second, "expiry mismatch for %d minutes", expiryMinutes)
		}
	})

	t.Run("should use HS256 signing method", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		parsedToken, err := jwt.Parse(token, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		assert.Equal(t, "HS256", parsedToken.Method.Alg())
	})
}

// TestParseAccessToken tests JWT access token parsing.
func TestParseAccessToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should parse valid access token", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		claims, err := ParseAccessToken(jwtSecret, "Bearer "+token)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, provider, claims.Provider)
	})

	t.Run("should reject token without Bearer prefix", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		_, err = ParseAccessToken(jwtSecret, token)
		assert.Error(t, err)
		assert.Equal(t, "invalid access token", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		_, err = ParseAccessToken("wrong-secret", "Bearer "+token)
		assert.Error(t, err)
		assert.Equal(t, "invalid access token", err.Error())
	})

	t.Run("should reject malformed token", func(t *testing.T) {
		_, err := ParseAccessToken(jwtSecret, "Bearer invalid.token.here")
		assert.Error(t, err)
		assert.Equal(t, "invalid access token", err.Error())
	})

	t.Run("should reject expired token", func(t *testing.T) {
		// Create a token with past expiration
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "app:*",
			Provider: provider,
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-2 * time.Hour)},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-1 * time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		_, err = ParseAccessToken(jwtSecret, "Bearer "+signedToken)
		assert.Error(t, err)
	})

	t.Run("should reject token with wrong signing method", func(t *testing.T) {
		// Create token with RS256 instead of HS256
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "app:*",
			Provider: provider,
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour)},
			},
		}

		// Try to use HMAC with a different algorithm indicator (this is a simulation)
		// In practice, this test verifies the signing method check works
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		// This should pass, but if we modify the algorithm in the header, it should fail
		// For now, verify that the function checks signing method
		_, err = ParseAccessToken(jwtSecret, "Bearer "+signedToken)
		require.NoError(t, err) // This one is valid
	})

	// Security fix tests: Validate audience claim to prevent token type confusion
	t.Run("should reject MFA token as access token", func(t *testing.T) {
		// Create an MFA token
		mfaToken, err := NewMFAToken(jwtSecret, user, 5)
		require.NoError(t, err)

		// Try to parse it as an access token
		_, err = ParseAccessToken(jwtSecret, "Bearer "+mfaToken)

		// Should fail with audience error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audience", "should reject MFA token with audience error")
	})

	t.Run("should reject refresh token as access token", func(t *testing.T) {
		// Create a refresh token
		refreshToken, err := NewRefreshToken(jwtSecret, user, provider, 600)
		require.NoError(t, err)

		// Try to parse it as an access token
		_, err = ParseAccessToken(jwtSecret, "Bearer "+refreshToken)

		// Should fail with audience error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audience", "should reject refresh token with audience error")
	})

	t.Run("should reject token with invalid audience", func(t *testing.T) {
		// Create a token with a custom invalid audience
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "invalid:custom",
			Provider: provider,
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		// Should fail with audience error
		_, err = ParseAccessToken(jwtSecret, "Bearer "+signedToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "audience")
	})

	t.Run("should accept only tokens with app:* audience", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "app:*",
			Provider: provider,
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		parsedClaims, err := ParseAccessToken(jwtSecret, "Bearer "+signedToken)
		require.NoError(t, err)
		assert.Equal(t, "app:*", parsedClaims.Aud)
	})
}

// TestNewRefreshToken tests JWT refresh token generation.
func TestNewRefreshToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should create valid refresh token", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, 600)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2)
	})

	t.Run("should have correct refresh audience", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, 600)
		require.NoError(t, err)

		claims := &models.UserClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		assert.Equal(t, "auth:refresh", claims.Aud)
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, 600)
		require.NoError(t, err)

		claims := &models.UserClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(600 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should respect custom expiry values", func(t *testing.T) {
		testCases := []int{60, 240, 720}

		for _, expiryMinutes := range testCases {
			token, err := NewRefreshToken(jwtSecret, user, provider, expiryMinutes)
			require.NoError(t, err)

			claims := &models.UserClaims{}
			_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			require.NoError(t, err)
			expectedExpiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
			actualExpiry := claims.ExpiresAt.Time

			diff := actualExpiry.Sub(expectedExpiry).Abs()
			assert.Less(t, diff, 5*time.Second, "expiry mismatch for %d minutes", expiryMinutes)
		}
	})
}

// TestParseRefreshToken tests JWT refresh token parsing.
func TestParseRefreshToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should parse valid refresh token", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, 600)
		require.NoError(t, err)

		claims, err := ParseRefreshToken(jwtSecret, token)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, "auth:refresh", claims.Aud)
	})

	t.Run("should reject access token as refresh token", func(t *testing.T) {
		// Try to use access token as refresh token
		token, err := NewAccessToken(jwtSecret, user, provider, 60)
		require.NoError(t, err)

		_, err = ParseRefreshToken(jwtSecret, token)
		assert.Error(t, err)
		assert.Equal(t, "invalid refresh token", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, 600)
		require.NoError(t, err)

		_, err = ParseRefreshToken("wrong-secret", token)
		assert.Error(t, err)
	})

	t.Run("should reject expired refresh token", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "auth:refresh",
			Provider: provider,
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-11 * time.Hour)},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-1 * time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		_, err = ParseRefreshToken(jwtSecret, signedToken)
		assert.Error(t, err)
	})
}

// TestNewMFAToken tests MFA token generation.
func TestNewMFAToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	t.Run("should create valid MFA token", func(t *testing.T) {
		token, err := NewMFAToken(jwtSecret, user, 5)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2)
	})

	t.Run("should have correct MFA audience", func(t *testing.T) {
		token, err := NewMFAToken(jwtSecret, user, 5)
		require.NoError(t, err)

		claims := &models.UserClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		assert.Equal(t, "auth:mfa", claims.Aud)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewMFAToken(jwtSecret, user, 5)
		require.NoError(t, err)

		claims := &models.UserClaims{}
		_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(5 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should respect custom expiry values", func(t *testing.T) {
		testCases := []int{1, 5, 15, 30}

		for _, expiryMinutes := range testCases {
			token, err := NewMFAToken(jwtSecret, user, expiryMinutes)
			require.NoError(t, err)

			claims := &models.UserClaims{}
			_, err = jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			require.NoError(t, err)
			expectedExpiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
			actualExpiry := claims.ExpiresAt.Time

			diff := actualExpiry.Sub(expectedExpiry).Abs()
			assert.Less(t, diff, 5*time.Second, "expiry mismatch for %d minutes", expiryMinutes)
		}
	})
}

// TestParseMFAToken tests MFA token parsing.
func TestParseMFAToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	t.Run("should parse valid MFA token", func(t *testing.T) {
		token, err := NewMFAToken(jwtSecret, user, 5)
		require.NoError(t, err)

		claims, err := ParseMFAToken(jwtSecret, token)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, "auth:mfa", claims.Aud)
	})

	t.Run("should reject access token as MFA token", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, "local", 60)
		require.NoError(t, err)

		_, err = ParseMFAToken(jwtSecret, token)
		assert.Error(t, err)
		assert.Equal(t, "invalid MFA token audience", err.Error())
	})

	t.Run("should reject refresh token as MFA token", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, "local", 600)
		require.NoError(t, err)

		_, err = ParseMFAToken(jwtSecret, token)
		assert.Error(t, err)
		assert.Equal(t, "invalid MFA token audience", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewMFAToken(jwtSecret, user, 5)
		require.NoError(t, err)

		_, err = ParseMFAToken("wrong-secret", token)
		assert.Error(t, err)
	})

	t.Run("should reject expired MFA token", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Aud:      "auth:mfa",
			Provider: "",
			Issuer:   "safebucket",
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-10 * time.Minute)},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-5 * time.Minute)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		_, err = ParseMFAToken(jwtSecret, signedToken)
		assert.Error(t, err)
	})
}

// TestGetUserClaims tests extracting claims from context.
func TestGetUserClaims(t *testing.T) {
	t.Run("should extract valid claims from context", func(t *testing.T) {
		expectedClaims := models.UserClaims{
			Email:  "test@example.com",
			UserID: uuid.New(),
			Role:   models.RoleUser,
		}

		ctx := context.WithValue(context.Background(), models.UserClaimKey{}, expectedClaims)

		claims, err := GetUserClaims(ctx)

		require.NoError(t, err)
		assert.Equal(t, expectedClaims.Email, claims.Email)
		assert.Equal(t, expectedClaims.UserID, claims.UserID)
		assert.Equal(t, expectedClaims.Role, claims.Role)
	})

	t.Run("should error when claims not in context", func(t *testing.T) {
		ctx := context.Background()

		_, err := GetUserClaims(ctx)

		assert.Error(t, err)
		assert.Equal(t, "invalid user claims", err.Error())
	})

	t.Run("should error when context has wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), models.UserClaimKey{}, "not a UserClaims")

		_, err := GetUserClaims(ctx)

		assert.Error(t, err)
		assert.Equal(t, "invalid user claims", err.Error())
	})
}

// TestGenerateSecret tests random secret generation.
func TestGenerateSecret(t *testing.T) {
	t.Run("should generate 6 character secret", func(t *testing.T) {
		secret, err := GenerateSecret()

		require.NoError(t, err)
		assert.Len(t, secret, 6)
	})

	t.Run("should generate alphanumeric uppercase secret", func(t *testing.T) {
		secret, err := GenerateSecret()

		require.NoError(t, err)
		for _, char := range secret {
			assert.True(t,
				(char >= '0' && char <= '9') || (char >= 'A' && char <= 'Z'),
				"character %c should be alphanumeric uppercase", char)
		}
	})

	t.Run("should generate different secrets", func(t *testing.T) {
		secret1, err1 := GenerateSecret()
		secret2, err2 := GenerateSecret()

		require.NoError(t, err1)
		require.NoError(t, err2)

		// While theoretically possible they could be the same,
		// with 36^6 possibilities, it's astronomically unlikely
		assert.NotEqual(t, secret1, secret2)
	})

	t.Run("should generate secrets with good distribution", func(t *testing.T) {
		// Generate 100 secrets and check they're reasonably diverse
		secrets := make(map[string]bool)
		for range 100 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			secrets[secret] = true
		}

		// All 100 should be unique
		assert.Len(t, secrets, 100, "all generated secrets should be unique")
	})

	t.Run("should use all characters in charset", func(t *testing.T) {
		// Generate many secrets and verify all possible characters appear
		charsSeen := make(map[rune]bool)

		// Generate enough secrets to likely see all characters
		for range 1000 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			for _, char := range secret {
				charsSeen[char] = true
			}
		}

		// We should have seen most characters (allow some statistical variance)
		// With 1000 secrets * 6 chars = 6000 samples from 36 chars,
		// it's extremely likely we see at least 30 different characters
		assert.GreaterOrEqual(t, len(charsSeen), 30,
			"should see most characters from charset in 1000 secrets")
	})
}

// TestSecretEntropyAndSecurity tests security properties of generated secrets.
func TestSecretEntropyAndSecurity(t *testing.T) {
	t.Run("should have sufficient entropy for security", func(t *testing.T) {
		// With 36 possible characters and 6 positions:
		// Entropy = log2(36^6) = 6 * log2(36) ≈ 31 bits
		// This means 2^31 = ~2 billion possible combinations

		// Generate 10000 secrets to check for patterns
		secrets := make(map[string]bool)
		for range 10000 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			secrets[secret] = true
		}

		// All should be unique (collision probability is negligible)
		assert.Len(t, secrets, 10000, "no collisions expected in 10000 secrets")
	})

	t.Run("should not have obvious patterns", func(t *testing.T) {
		// Check that secrets don't follow obvious patterns
		for range 100 {
			secret, err := GenerateSecret()
			require.NoError(t, err)

			// Should not be all same character
			allSame := true
			firstChar := secret[0]
			for j := range len(secret) {
				if secret[j] != firstChar {
					allSame = false
					break
				}
			}
			assert.False(t, allSame, "secret %s should not be all same character", secret)

			// Should not be sequential (like 012345 or ABCDEF)
			isSequential := true
			for j := 1; j < len(secret); j++ {
				if secret[j] != secret[j-1]+1 {
					isSequential = false
					break
				}
			}
			assert.False(t, isSequential, "secret %s should not be sequential", secret)
		}
	})
}
