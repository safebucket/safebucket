package services

import (
	c "api/internal/common"
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
	r.Get("/", c.GetListHandler(s.GetUserList))
	r.With(c.Validate[models.UserCreateBody]).Post("/", c.CreateHandler(s.CreateUser))
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", c.GetOneHandler(s.GetUser))
		r.With(c.Validate[models.UserUpdateBody]).Patch("/", c.UpdateHandler(s.UpdateUser))
		r.Delete("/", c.DeleteHandler(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(body models.UserCreateBody) (models.User, error) {
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

func (s UserService) GetUser(id string) (models.User, error) {
	var user models.User

	_, err := uuid.Parse(id)
	if err != nil {
		return models.User{}, errors.New("invalid ID")
	}

	result := s.DB.Where("id = ?", id).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(id string, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: id}
	_, err := uuid.Parse(id)
	if err != nil {
		return models.User{}, errors.New("invalid ID")
	}
	result := s.DB.Model(&user).Updates(body)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) DeleteUser(id string) error {
	result := s.DB.Where("id = ?", id).Delete(&models.User{})
	_, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid ID")
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	} else {
		return nil
	}
}
