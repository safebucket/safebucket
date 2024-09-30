package helpers

import (
	"api/internal/models"
	"errors"
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
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
		Aud:    "app:*", // Todo: make it a list ==> delete aud
		Issuer: "SafeBucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * 10)}, // TODO: make it configurable
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return accessToken.SignedString([]byte(jwtSecret))
}

func ParseAccessToken(jwtSecret string, accessToken string) (*models.UserClaims, error) {
	parsedAccessToken, err := jwt.ParseWithClaims(
		accessToken,
		&models.UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		},
	)
	if parsedAccessToken.Claims.(*models.UserClaims).Aud != "app:*" {
		return &models.UserClaims{}, errors.New("invalid access token")
	}
	return parsedAccessToken.Claims.(*models.UserClaims), err
}

func NewRefreshToken(jwtSecret string, user *models.User) (string, error) {
	claims := models.UserClaims{
		Email:  user.Email,
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
	parsedAccessToken, err := jwt.ParseWithClaims(
		refreshToken,
		&models.UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		},
	)
	if parsedAccessToken.Claims.(models.UserClaims).Aud != "auth:refresh" {
		return models.UserClaims{}, errors.New("invalid refresh token")
	}
	return parsedAccessToken.Claims.(models.UserClaims), err
}
