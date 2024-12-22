package services

import (
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	a "api/internal/rbac"
	"api/internal/rbac/roles"
	"api/internal/rbac/types"
	"errors"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	DB      *gorm.DB
	JWTConf models.JWTConfiguration
	E       *casbin.Enforcer
}

func (s UserService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", handlers.GetListHandler(s.GetUserList))
	r.With(h.Validate[models.UserCreateBody]).Post("/", handlers.CreateHandler(s.CreateUser))
	r.Route("/{id}", func(r chi.Router) {
		r.With(a.Authenticate(s.JWTConf)).Get("/", handlers.GetOneHandler(s.GetUser))
		r.With(a.Authenticate(s.JWTConf)).With(h.Validate[models.UserUpdateBody]).Patch("/", handlers.UpdateHandler(s.UpdateUser))
		r.With(a.Authenticate(s.JWTConf)).Delete("/", handlers.DeleteHandler(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(u *models.UserClaims, body models.UserCreateBody) (models.User, error) {
	newUser := models.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
	}
	result := s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected == 0 {

		hash, err := h.CreateHash(body.Password)
		if err != nil {
			return models.User{}, errors.New("can not create hash password")
		}
		newUser.HashedPassword = hash
		s.DB.Create(&newUser)
		err = roles.AddUserToRoleUser(s.E, newUser)
		if err != nil {
			return models.User{}, err
		}
		return newUser, nil
	} else {
		return models.User{}, errors.New("user already exists, try to reset your password")
	}

}

func (s UserService) GetUserList(u *models.UserClaims) []models.User {
	var users []models.User
	s.DB.Find(&users)
	return users
}

func (s UserService) GetUser(u *models.UserClaims, id uuid.UUID) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", id).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(u *models.UserClaims, id uuid.UUID, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: id}
	result := s.DB.Model(&user).Updates(body)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) DeleteUser(u *models.UserClaims, id uuid.UUID) error {
	result := s.DB.Where("id = ?", id).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	} else {
		return nil
	}
}
