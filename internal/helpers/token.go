package helpers

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"api/internal/configuration"
	"api/internal/models"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// tokenConfig holds configuration for creating a specific token type.
type tokenConfig struct {
	audience      string
	provider      string
	mfa           *bool // nil = don't set (defaults to false), otherwise set to this value
	expiryMinutes int
	challengeID   *uuid.UUID
}

func boolPtr(b bool) *bool {
	return &b
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

	if config.mfa != nil {
		claims.MFA = *config.mfa
	}

	if config.challengeID != nil {
		claims.ChallengeID = config.challengeID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ParseToken parses and validates a JWT token without audience validation.
// It validates signature, expiry, and issuer only.
// Audience validation is delegated to the AudienceValidate middleware for route-specific rules.
// The requireBearer parameter controls whether the "Bearer " prefix is required.
func ParseToken(jwtSecret string, tokenString string, requireBearer bool) (models.UserClaims, error) {
	if requireBearer {
		if !strings.HasPrefix(tokenString, "Bearer ") {
			return models.UserClaims{}, errors.New("invalid token")
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
		return models.UserClaims{}, errors.New("invalid token")
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
		mfa:           boolPtr(user.HasMFAEnabled()),
		expiryMinutes: configuration.AccessTokenExpiry,
	})
}

func NewRefreshToken(jwtSecret string, user *models.User, provider string) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      configuration.AudienceRefreshToken,
		provider:      provider,
		mfa:           boolPtr(user.HasMFAEnabled()),
		expiryMinutes: configuration.RefreshTokenExpiry,
	})
}

// ParseRefreshToken validates and parses a refresh token.
// Returns error if token is invalid or has wrong audience.
func ParseRefreshToken(jwtSecret string, refreshToken string) (models.UserClaims, error) {
	claims, err := ParseToken(jwtSecret, refreshToken, false)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid refresh token")
	}

	if claims.Aud != configuration.AudienceRefreshToken {
		return models.UserClaims{}, errors.New("invalid refresh token audience")
	}

	return claims, nil
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

// NewRestrictedAccessToken creates a restricted access token for MFA flows.
// This token grants limited access: only MFA device management and verification endpoints.
// Used for both login MFA and password reset MFA flows.
// Audience: "auth:mfa:login" or "auth:mfa:password-reset".
// For password reset flow, challengeID should be provided to link the token to the challenge.
func NewRestrictedAccessToken(
	jwtSecret string,
	user *models.User,
	audience string,
	mfaVerified bool,
	challengeID *uuid.UUID,
) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      audience,
		provider:      string(user.ProviderType),
		mfa:           boolPtr(mfaVerified),
		expiryMinutes: configuration.MFATokenExpiry,
		challengeID:   challengeID,
	})
}
