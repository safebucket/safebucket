package services

import (
	c "api/internal/common"
	h "api/internal/helpers"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
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

func (s UserService) GetUser(id uint) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", id).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(id uint, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: id}
	result := s.DB.Model(&user).Updates(body)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (s UserService) DeleteUser(id uint) error {
	result := s.DB.Where("id = ?", id).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	} else {
		return nil
	}
}
