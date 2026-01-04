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

func NewAccessToken(jwtSecret string, user *models.User, provider string, expiryMinutes int) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Aud:      "app:*",
		Provider: provider,
		Issuer:   configuration.AppName,
		MFA:      user.HasMFAEnabled(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * time.Duration(expiryMinutes))},
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return accessToken.SignedString([]byte(jwtSecret))
}

func ParseAccessToken(jwtSecret string, accessToken string) (models.UserClaims, error) {
	if !strings.HasPrefix(accessToken, "Bearer ") {
		return models.UserClaims{}, errors.New("invalid access token")
	}
	accessToken = strings.TrimPrefix(accessToken, "Bearer ")
	claims := &models.UserClaims{}

	_, err := jwt.ParseWithClaims(
		accessToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		},
	)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid access token")
	}

	if claims.Aud != "app:*" {
		return models.UserClaims{}, errors.New("invalid access token audience")
	}

	return *claims, nil
}

func NewRefreshToken(jwtSecret string, user *models.User, provider string, expiryMinutes int) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Aud:      "auth:refresh",
		Issuer:   configuration.AppName,
		Provider: provider,
		MFA:      user.HasMFAEnabled(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * time.Duration(expiryMinutes))},
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return refreshToken.SignedString([]byte(jwtSecret))
}

func ParseRefreshToken(jwtSecret string, refreshToken string) (models.UserClaims, error) {
	claims := &models.UserClaims{}

	_, err := jwt.ParseWithClaims(
		refreshToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		},
	)

	if claims.Aud != "auth:refresh" {
		return models.UserClaims{}, errors.New("invalid refresh token")
	}
	return *claims, err
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
func NewMFAToken(jwtSecret string, user *models.User, expiryMinutes int) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Aud:      "auth:mfa",
		Issuer:   configuration.AppName,
		Provider: string(user.ProviderType),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * time.Duration(expiryMinutes))},
		},
	}
	mfaToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return mfaToken.SignedString([]byte(jwtSecret))
}

// ParseMFAToken validates and parses an MFA token.
// Returns the user claims if the token is valid and has the correct audience.
func ParseMFAToken(jwtSecret string, mfaToken string) (models.UserClaims, error) {
	claims := &models.UserClaims{}

	_, err := jwt.ParseWithClaims(
		mfaToken,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		},
	)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid MFA token")
	}

	if claims.Aud != "auth:mfa" {
		return models.UserClaims{}, errors.New("invalid MFA token audience")
	}

	return *claims, nil
}
