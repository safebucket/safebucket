package services

import (
	c "api/internal/common"
	"api/internal/models"
	"errors"
	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"time"
)

type AuthService struct {
	DB      *gorm.DB
	JWTConf models.JWTConfiguration
}

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(c.Validate[models.AuthLogin]).Post("/login", c.CreateHandler(s.Login))
	r.With(c.Validate[models.AuthVerify]).Post("/verify", c.CreateHandler(s.Verify))
	r.With(c.Validate[models.AuthVerify]).Post("/refresh", c.CreateHandler(s.Refresh))
	return r
}

func (s AuthService) Login(body models.AuthLogin) (models.AuthLoginResponse, error) {
	searchUser := models.User{
		Email: body.Email,
	}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 1 {
		match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
		if err != nil || match == false {
			return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
		}

		token, err := s.NewAccessToken(&searchUser)

		if err != nil {
			return models.AuthLoginResponse{}, errors.New("failed to generate new access token")
		}

		return models.AuthLoginResponse{Token: token}, nil
	}
	return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
}

func (s AuthService) Verify(body models.AuthVerify) (any, error) {
	data, err := s.ParseAccessToken(body.Token)
	return data, err
}

func (s AuthService) NewAccessToken(user *models.User) (string, error) {
	claims := models.UserClaims{
		Email:  user.Email,
		Aud:    "app:*",
		Issuer: "SafeBucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute * 10)}, // TODO: make it configurable
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return accessToken.SignedString([]byte(s.JWTConf.Secret))
}

func (s AuthService) ParseAccessToken(accessToken string) (models.UserClaims, error) {
	parsedAccessToken, err := jwt.ParseWithClaims(accessToken, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.JWTConf.Secret), nil
	})
	if parsedAccessToken.Claims.(models.UserClaims).Aud != "app:*" {
		return models.UserClaims{}, errors.New("invalid access token")
	}
	return parsedAccessToken.Claims.(models.UserClaims), err
}

func (s AuthService) NewRefreshToken(user *models.User) (string, error) {
	claims := models.UserClaims{
		Email:  user.Email,
		Aud:    "auth:refresh",
		Issuer: "SafeBucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour * 10)}, // TODO: make it configurable
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return refreshToken.SignedString([]byte(s.JWTConf.Secret))
}

func (s AuthService) ParseRefreshToken(refreshToken string) (models.UserClaims, error) {
	parsedAccessToken, err := jwt.ParseWithClaims(refreshToken, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.JWTConf.Secret), nil
	})
	if parsedAccessToken.Claims.(models.UserClaims).Aud != "auth:refresh" {
		return models.UserClaims{}, errors.New("invalid refresh token")
	}
	return parsedAccessToken.Claims.(models.UserClaims), err
}

func (s AuthService) Refresh(body models.AuthRefresh) (models.AuthRefreshResponse, error) {
	RefreshToken, err := s.ParseRefreshToken(body.RefreshToken)
	if err != nil {
		return models.AuthRefreshResponse{}, err
	}
	accessToken, err := s.NewAccessToken(&models.User{Email: RefreshToken.Email})
	return models.AuthRefreshResponse{AccessToken: accessToken}, err
}
