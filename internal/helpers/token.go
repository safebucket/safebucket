package helpers

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"api/internal/models"

	"api/internal/configuration"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

// tokenConfig holds configuration for creating a specific token type
type tokenConfig struct {
	audience      string
	provider      string
	mfaEnabled    bool
	includeMFA    bool // Whether to populate the MFA field
	expiryMinutes int  // From configuration constants
}

// parseTokenConfig holds configuration for parsing a specific token type
type parseTokenConfig struct {
	tokenString      string
	expectedAudience string
	requireBearer    bool
	errorMessage     string
	audienceError    string
}

// createToken is a generic helper for creating JWT tokens with specified configuration.
// This private function consolidates the common token creation logic used by all public
// token creation functions (NewAccessToken, NewRefreshToken, etc.).
func createToken(jwtSecret string, user *models.User, config tokenConfig) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Aud:      config.audience,
		Issuer:   configuration.AppName,
		Provider: config.provider,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * time.Duration(config.expiryMinutes))},
		},
	}

	// Only populate MFA field for access and refresh tokens
	if config.includeMFA {
		claims.MFA = config.mfaEnabled
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// parseToken is a generic helper for parsing and validating JWT tokens.
// This private function consolidates the common token parsing logic used by all public
// token parsing functions (ParseAccessToken, ParseRefreshToken, etc.).
func parseToken(jwtSecret string, config parseTokenConfig) (models.UserClaims, error) {
	tokenString := config.tokenString

	// Handle Bearer prefix if required
	if config.requireBearer {
		if !strings.HasPrefix(tokenString, "Bearer ") {
			return models.UserClaims{}, errors.New(config.errorMessage)
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	claims := &models.UserClaims{}

	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		},
	)
	if err != nil {
		return models.UserClaims{}, errors.New(config.errorMessage)
	}

	// Validate audience to prevent token type confusion attacks
	if claims.Aud != config.expectedAudience {
		return models.UserClaims{}, errors.New(config.audienceError)
	}

	return *claims, nil
}

func CreateHash(password string) (string, error) {
	argonParams := argon2id.Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  32,
		KeyLength:   32,
	}
	hash, err := argon2id.CreateHash(password, &argonParams)
	if err != nil {
		return "", errors.New("can not create hash password")
	}

	return hash, nil
}

func NewAccessToken(jwtSecret string, user *models.User, provider string) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceAccessToken,
		provider:      provider,
		mfaEnabled:    user.HasMFAEnabled(),
		includeMFA:    true,
		expiryMinutes: configuration.AccessTokenExpiry,
	})
}

func ParseAccessToken(jwtSecret string, accessToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      accessToken,
		expectedAudience: configuration.AudienceAccessToken,
		requireBearer:    true,
		errorMessage:     "invalid access token",
		audienceError:    "invalid access token audience",
	})
}

func NewRefreshToken(jwtSecret string, user *models.User, provider string) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceRefreshToken,
		provider:      provider,
		mfaEnabled:    user.HasMFAEnabled(),
		includeMFA:    true,
		expiryMinutes: configuration.RefreshTokenExpiry,
	})
}

func ParseRefreshToken(jwtSecret string, refreshToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      refreshToken,
		expectedAudience: configuration.AudienceRefreshToken,
		requireBearer:    false,
		errorMessage:     "invalid refresh token",
		audienceError:    "invalid refresh token audience",
	})
}

func GetUserClaims(c context.Context) (models.UserClaims, error) {
	value, ok := c.Value(models.UserClaimKey{}).(models.UserClaims)
	if !ok {
		return models.UserClaims{}, errors.New("invalid user claims")
	}
	return value, nil
}

func GenerateSecret() (string, error) {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const secretLength = 6
	secret := make([]byte, secretLength)
	for i := range secret {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		secret[i] = charset[n.Int64()]
	}
	return string(secret), nil
}

// NewMFAToken creates a short-lived JWT token for MFA verification during login.
// This token is issued after successful password authentication but before MFA verification.
// It can only be used to complete the MFA verification step.
func NewMFAToken(jwtSecret string, user *models.User) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceMFALoginToken,
		provider:      string(user.ProviderType),
		mfaEnabled:    false,
		includeMFA:    false,
		expiryMinutes: configuration.MFATokenExpiry,
	})
}

// ParseMFAToken validates and parses an MFA token.
// Returns the user claims if the token is valid and has the correct audience.
func ParseMFAToken(jwtSecret string, mfaToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      mfaToken,
		expectedAudience: configuration.AudienceMFALoginToken,
		requireBearer:    false,
		errorMessage:     "invalid MFA token",
		audienceError:    "invalid MFA token audience",
	})
}

// NewPasswordResetMFAToken creates an MFA token for password reset flow.
// Uses audience "auth:mfa:password-reset" to distinguish from login MFA.
// The challenge ID is stored separately in cache, keyed by user ID.
func NewPasswordResetMFAToken(jwtSecret string, user *models.User) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceMFAPasswordResetToken,
		provider:      string(user.ProviderType),
		mfaEnabled:    false,
		includeMFA:    false,
		expiryMinutes: configuration.PasswordResetMFATokenExpiry,
	})
}

// ParsePasswordResetMFAToken validates and parses a password reset MFA token.
// Returns the user claims if the token is valid and has the correct audience.
func ParsePasswordResetMFAToken(jwtSecret string, mfaToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      mfaToken,
		expectedAudience: configuration.AudienceMFAPasswordResetToken,
		requireBearer:    false,
		errorMessage:     "invalid password reset MFA token",
		audienceError:    "invalid password reset MFA token audience",
	})
}

// NewPasswordResetCompletionToken creates a token that authorizes password change.
// Issued after successful code verification (and MFA if enabled).
// Audience: "auth:password-reset"
func NewPasswordResetCompletionToken(jwtSecret string, user *models.User) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudiencePasswordResetCompletion,
		provider:      string(user.ProviderType),
		mfaEnabled:    false,
		includeMFA:    false,
		expiryMinutes: configuration.PasswordResetCompletionExpiry,
	})
}

// ParsePasswordResetCompletionToken validates a password reset completion token.
// Returns claims if valid, error otherwise.
func ParsePasswordResetCompletionToken(jwtSecret string, token string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      token,
		expectedAudience: configuration.AudiencePasswordResetCompletion,
		requireBearer:    false,
		errorMessage:     "invalid password reset token",
		audienceError:    "invalid password reset token audience",
	})
}
