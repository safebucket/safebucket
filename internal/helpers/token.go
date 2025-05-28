package helpers

import (
	"api/internal/models"
	"context"
	"errors"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
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

func NewAccessToken(jwtSecret string, user *models.User) (string, error) {
	claims := models.UserClaims{
		Email:  user.Email,
		UserID: user.ID,
		Aud:    "app:*", // Todo: make it a list ==> delete aud
		Issuer: "safebucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * 60)}, // TODO: make it configurable
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
			return []byte(jwtSecret), nil
		},
	)

	if err != nil {
		return models.UserClaims{}, errors.New("invalid access token")
	}

	if claims.Aud != "app:*" {
		return models.UserClaims{}, errors.New("invalid scope for this access token")
	}

	return *claims, err
}

func NewRefreshToken(jwtSecret string, user *models.User) (string, error) {
	claims := models.UserClaims{
		Email:  user.Email,
		UserID: user.ID,
		Aud:    "auth:refresh",
		Issuer: "SafeBucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour * 10)},
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
