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

// tokenConfig holds configuration for creating a specific token type.
type tokenConfig struct {
	audience      string
	provider      string
	mfa           *bool // nil = don't set (defaults to false), otherwise set to this value
	expiryMinutes int   // From configuration constants
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool {
	return &b
}

// parseTokenConfig holds configuration for parsing a specific token type.
type parseTokenConfig struct {
	tokenString      string
	allowedAudiences []string // One or more allowed audience values
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

	if config.mfa != nil {
		claims.MFA = *config.mfa
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
	audienceValid := false
	for _, allowed := range config.allowedAudiences {
		if claims.Aud == allowed {
			audienceValid = true
			break
		}
	}
	if !audienceValid {
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
		mfa:           boolPtr(user.HasMFAEnabled()),
		expiryMinutes: configuration.AccessTokenExpiry,
	})
}

func ParseAccessToken(jwtSecret string, accessToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      accessToken,
		allowedAudiences: []string{configuration.AudienceAccessToken},
		requireBearer:    true,
		errorMessage:     "invalid access token",
		audienceError:    "invalid access token audience",
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

func ParseRefreshToken(jwtSecret string, refreshToken string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      refreshToken,
		allowedAudiences: []string{configuration.AudienceRefreshToken},
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

// NewRestrictedAccessToken creates a restricted access token for MFA flows.
// This token grants limited access: only MFA device management and verification endpoints.
// Used for both login MFA and password reset MFA flows.
// Audience: "auth:mfa:login" or "auth:mfa:password-reset".
func NewRestrictedAccessToken(jwtSecret string, user *models.User, audience string, mfaVerified bool) (string, error) {
	return createToken(jwtSecret, user, tokenConfig{
		audience:      audience,
		provider:      string(user.ProviderType),
		mfa:           boolPtr(mfaVerified),
		expiryMinutes: configuration.MFATokenExpiry,
	})
}

// ParseRestrictedAccessToken validates and parses a restricted access token.
// Returns the user claims if the token is valid and has one of the allowed restricted audiences:
// - auth:mfa:login (login MFA flow)
// - auth:mfa:password-reset (password reset MFA flow)
// Accepts Bearer prefix for consistency with regular access tokens.
// Note: Callers should perform additional audience checks if flow-specific validation is needed.
func ParseRestrictedAccessToken(jwtSecret string, token string) (models.UserClaims, error) {
	return parseToken(jwtSecret, parseTokenConfig{
		tokenString:      token,
		allowedAudiences: []string{configuration.AudienceMFALogin, configuration.AudienceMFAReset},
		requireBearer:    true,
		errorMessage:     "invalid restricted access token",
		audienceError:    "invalid restricted access token audience",
	})
}
