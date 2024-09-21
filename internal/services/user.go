package services

import (
	c "api/internal/common"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func (s UserService) RoutesV2() chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.GetListHandlerV2(s.GetUserList))
	r.With(c.Validate[models.UserCreateBody]).Post("/", c.CreateHandlerV2(s.CreateUser))

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", c.GetOneHandlerV2(s.GetUser))
		r.With(c.Validate[models.UserUpdateBody]).Patch("/", c.UpdateHandlerV2(s.UpdateUser))
		r.Delete("/", c.DeleteHandlerV2(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(body models.UserCreateBody) error {
	newUser := models.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
		Email:     body.Email,
	}
	result := s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected == 0 {
		s.DB.Create(&newUser)
		return nil
	} else {
		return errors.New("user already exists")
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
