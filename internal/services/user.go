package services

import (
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func (s UserService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", handlers.GetListHandler(s.GetUserList))
	r.With(h.Validate[models.UserCreateBody]).Post("/", handlers.CreateHandler(s.CreateUser))
	r.Route("/{id0}", func(r chi.Router) {
		r.Get("/", handlers.GetOneHandler(s.GetUser))
		r.With(h.Validate[models.UserUpdateBody]).Patch("/", handlers.UpdateHandler(s.UpdateUser))
		r.Delete("/", handlers.DeleteHandler(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(_ uuid.UUIDs, body models.UserCreateBody) (models.User, error) {
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

		return newUser, nil
	} else {
		return models.User{}, errors.New("user already exists, try to reset your password")
	}
}

func (s UserService) GetUserList() []models.User {
	var users []models.User
	s.DB.Find(&users)
	return users
}

func (s UserService) GetUser(ids uuid.UUIDs) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", ids[0]).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(ids uuid.UUIDs, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: ids[0]}
	result := s.DB.Model(&user).Updates(body)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) DeleteUser(ids uuid.UUIDs) error {
	result := s.DB.Where("id = ?", ids[0]).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	} else {
		return nil
	}
}
