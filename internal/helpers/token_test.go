package helpers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNewAccessToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should create valid access token", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		// JWE compact serialization has 5 parts separated by 4 dots.
		assert.True(t, strings.Count(token, ".") == 4, "JWE should have 5 parts separated by dots")
	})

	t.Run("should have correct claims", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, provider, claims.Provider)
		assert.Equal(t, configuration.AppName, claims.Issuer)
		assert.Equal(t, "app:*", claims.Audience[0])
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(60 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		// Allow 5 second tolerance for test execution time.
		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should use configuration constant for expiry", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(time.Duration(configuration.AccessTokenExpiry) * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should use HS256 signing method", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		jws, err := decryptJWE(jwtSecret, token)
		require.NoError(t, err)

		parsedToken, err := jwt.Parse(jws, func(_ *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		require.NoError(t, err)
		assert.Equal(t, "HS256", parsedToken.Method.Alg())
	})

	t.Run("should have typed JTI with access prefix", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "test-sid")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(claims.ID, "access:"), "JTI should have access: prefix")
		assert.Equal(t, "test-sid", claims.SID)
	})
}

func TestParseToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should parse valid token with Bearer prefix", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, "Bearer "+token, true)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, "app:*", claims.Audience[0]) // Audience is in claims, not validated
	})

	t.Run("should parse valid token without Bearer prefix when not required", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, "auth:refresh", claims.Audience[0])
	})

	t.Run("should reject token without Bearer prefix when required", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		_, err = ParseToken(jwtSecret, token, true)
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject malformed token", func(t *testing.T) {
		_, err := ParseToken(jwtSecret, "Bearer invalid.token.here", true)
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		_, err = ParseToken("wrong-secret", "Bearer "+token, true)
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject expired token", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Provider: provider,
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{"app:*"},
				Issuer:    configuration.AppName,
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-2 * time.Hour)},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-1 * time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		_, err = ParseToken(jwtSecret, "Bearer "+signedToken, true)
		assert.Error(t, err)
	})

	t.Run("should parse token with any single audience", func(t *testing.T) {
		accessToken, _ := NewAccessToken(jwtSecret, user, provider, "")
		refreshToken, _ := NewRefreshToken(jwtSecret, user, provider, "")
		mfaToken, _ := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)

		claims1, err1 := ParseToken(jwtSecret, "Bearer "+accessToken, true)
		claims2, err2 := ParseToken(jwtSecret, refreshToken, false)
		claims3, err3 := ParseToken(jwtSecret, "Bearer "+mfaToken, true)

		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)

		assert.Equal(t, "app:*", claims1.Audience[0])
		assert.Equal(t, "auth:refresh", claims2.Audience[0])
		assert.Equal(t, configuration.AudienceMFALogin, claims3.Audience[0])
	})

	t.Run("should reject token with multiple audiences", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Provider: provider,
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{"app:*", "auth:refresh"},
				Issuer:    configuration.AppName,
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(1 * time.Hour)},
			},
		}
		jws, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
		require.NoError(t, err)
		jweToken, err := encryptJWE(jwtSecret, jws)
		require.NoError(t, err)

		_, err = ParseToken(jwtSecret, "Bearer "+jweToken, true)
		assert.Error(t, err)
		assert.Equal(t, "invalid token audience", err.Error())
	})

	t.Run("should reject token with no audience", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Provider: provider,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    configuration.AppName,
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(1 * time.Hour)},
			},
		}
		jws, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
		require.NoError(t, err)
		jweToken, err := encryptJWE(jwtSecret, jws)
		require.NoError(t, err)

		_, err = ParseToken(jwtSecret, "Bearer "+jweToken, true)
		assert.Error(t, err)
		assert.Equal(t, "invalid token audience", err.Error())
	})
}

func TestNewRefreshToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should create valid refresh token", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 4, "JWE should have 5 parts separated by dots")
	})

	t.Run("should have correct refresh audience", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.Equal(t, "auth:refresh", claims.Audience[0])
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(600 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should use configuration constant for expiry", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(time.Duration(configuration.RefreshTokenExpiry) * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should have typed JTI with refresh prefix", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "test-sid")
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(claims.ID, "refresh:"), "JTI should have refresh: prefix")
		assert.Equal(t, "test-sid", claims.SID)
	})
}

func TestParseRefreshToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}
	provider := "local"

	t.Run("should parse valid refresh token", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		claims, err := ParseRefreshToken(jwtSecret, token)

		require.NoError(t, err)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, "auth:refresh", claims.Audience[0])
	})

	t.Run("should reject access token as refresh token", func(t *testing.T) {
		token, err := NewAccessToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		_, err = ParseRefreshToken(jwtSecret, token)
		assert.Error(t, err)
		assert.Equal(t, "invalid refresh token audience", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewRefreshToken(jwtSecret, user, provider, "")
		require.NoError(t, err)

		_, err = ParseRefreshToken("wrong-secret", token)
		assert.Error(t, err)
	})

	t.Run("should reject expired refresh token", func(t *testing.T) {
		claims := models.UserClaims{
			Email:    user.Email,
			UserID:   user.ID,
			Role:     user.Role,
			Provider: provider,
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{"auth:refresh"},
				Issuer:    configuration.AppName,
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

func TestNewRestrictedAccessToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	user := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	t.Run("should create valid restricted access token", func(t *testing.T) {
		token, err := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 4, "JWE should have 5 parts separated by dots")
	})

	t.Run("should have correct auth:mfa audience", func(t *testing.T) {
		token, err := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.Equal(t, configuration.AudienceMFALogin, claims.Audience[0])
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.ID, claims.UserID)
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(5 * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should use configuration constant for expiry", func(t *testing.T) {
		token, err := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(time.Duration(configuration.MFATokenExpiry) * time.Minute)
		actualExpiry := claims.ExpiresAt.Time

		diff := actualExpiry.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})

	t.Run("should have empty JTI and no SID for restricted tokens", func(t *testing.T) {
		token, err := NewRestrictedAccessToken(jwtSecret, user, configuration.AudienceMFALogin, false, nil)
		require.NoError(t, err)

		claims, err := ParseToken(jwtSecret, token, false)

		require.NoError(t, err)
		assert.Empty(t, claims.ID, "restricted tokens should have empty JTI")
		assert.Empty(t, claims.SID, "restricted tokens should have no SID")
	})
}

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

		assert.NotEqual(t, secret1, secret2)
	})

	t.Run("should generate secrets with good distribution", func(t *testing.T) {
		secrets := make(map[string]bool)
		for range 100 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			secrets[secret] = true
		}

		assert.Len(t, secrets, 100, "all generated secrets should be unique")
	})

	t.Run("should use all characters in charset", func(t *testing.T) {
		charsSeen := make(map[rune]bool)

		for range 1000 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			for _, char := range secret {
				charsSeen[char] = true
			}
		}

		assert.GreaterOrEqual(t, len(charsSeen), 30,
			"should see most characters from charset in 1000 secrets")
	})
}

func TestNewShareAccessToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	shareID := uuid.New()

	t.Run("should create valid share token", func(t *testing.T) {
		token, err := NewShareAccessToken(jwtSecret, shareID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, 4, strings.Count(token, "."), "JWE should have 5 parts separated by dots")
	})

	t.Run("should have correct claims", func(t *testing.T) {
		token, err := NewShareAccessToken(jwtSecret, shareID)
		require.NoError(t, err)

		claims, err := ParseShareToken(jwtSecret, token)

		require.NoError(t, err)
		assert.Equal(t, shareID, claims.ShareID)
		assert.Equal(t, configuration.AppName, claims.Issuer)
		assert.Equal(t, configuration.AudienceShareAccess, claims.Audience[0])
	})

	t.Run("should expire in configured minutes", func(t *testing.T) {
		token, err := NewShareAccessToken(jwtSecret, shareID)
		require.NoError(t, err)

		claims, err := ParseShareToken(jwtSecret, token)

		require.NoError(t, err)
		expectedExpiry := time.Now().Add(
			time.Duration(configuration.ShareTokenExpiry) * time.Minute,
		)
		diff := claims.ExpiresAt.Time.Sub(expectedExpiry).Abs()
		assert.Less(t, diff, 5*time.Second)
	})
}

func TestParseShareToken(t *testing.T) {
	jwtSecret := "test-secret-key"
	shareID := uuid.New()

	t.Run("should parse valid share token", func(t *testing.T) {
		token, err := NewShareAccessToken(jwtSecret, shareID)
		require.NoError(t, err)

		claims, err := ParseShareToken(jwtSecret, token)

		require.NoError(t, err)
		assert.Equal(t, shareID, claims.ShareID)
		assert.Equal(t, configuration.AudienceShareAccess, claims.Audience[0])
	})

	t.Run("should reject empty token", func(t *testing.T) {
		_, err := ParseShareToken(jwtSecret, "")
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject malformed token", func(t *testing.T) {
		_, err := ParseShareToken(jwtSecret, "invalid.token.here")
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject token with wrong secret", func(t *testing.T) {
		token, err := NewShareAccessToken(jwtSecret, shareID)
		require.NoError(t, err)

		_, err = ParseShareToken("wrong-secret", token)
		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	t.Run("should reject expired share token", func(t *testing.T) {
		claims := models.ShareClaims{
			ShareID: shareID,
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{configuration.AudienceShareAccess},
				Issuer:    configuration.AppName,
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Add(-2 * time.Hour)},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(-1 * time.Hour)},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(jwtSecret))
		require.NoError(t, err)

		_, err = ParseShareToken(jwtSecret, signedToken)
		assert.Error(t, err)
	})

	t.Run("should reject user token", func(t *testing.T) {
		user := &models.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Role:  models.RoleUser,
		}
		userToken, err := NewAccessToken(jwtSecret, user, "local", "")
		require.NoError(t, err)

		_, err = ParseShareToken(jwtSecret, userToken)
		assert.Error(t, err)
		assert.Equal(t, "invalid token audience", err.Error())
	})

	t.Run("should reject token with wrong audience", func(t *testing.T) {
		claims := models.ShareClaims{
			ShareID: shareID,
			RegisteredClaims: jwt.RegisteredClaims{
				Audience:  jwt.ClaimStrings{"wrong:audience"},
				Issuer:    configuration.AppName,
				IssuedAt:  &jwt.NumericDate{Time: time.Now()},
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(1 * time.Hour)},
			},
		}
		jws, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
		require.NoError(t, err)
		jweToken, err := encryptJWE(jwtSecret, jws)
		require.NoError(t, err)

		_, err = ParseShareToken(jwtSecret, jweToken)
		assert.Error(t, err)
		assert.Equal(t, "invalid token audience", err.Error())
	})
}

func TestSecretEntropyAndSecurity(t *testing.T) {
	t.Run("should have sufficient entropy for security", func(t *testing.T) {
		secrets := make(map[string]bool)
		for range 10000 {
			secret, err := GenerateSecret()
			require.NoError(t, err)
			secrets[secret] = true
		}

		assert.Len(t, secrets, 10000, "no collisions expected in 10000 secrets")
	})

	t.Run("should not have obvious patterns", func(t *testing.T) {
		for range 100 {
			secret, err := GenerateSecret()
			require.NoError(t, err)

			allSame := true
			firstChar := secret[0]
			for j := range len(secret) {
				if secret[j] != firstChar {
					allSame = false
					break
				}
			}
			assert.False(t, allSame, "secret %s should not be all same character", secret)

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
