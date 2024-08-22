package api

import (
	c "api/internal/common"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func (ur UserRepo) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.GetListHandler[models.User](ur))
	r.With(c.Validate[models.User]).Post("/", c.CreateHandler[models.User](ur))

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", c.GetOneHandler[models.User](ur))
		r.Delete("/", c.DeleteHandler[models.User](ur))
	})
	return r
}

type UserRepo struct {
	DB *gorm.DB
}

func (ur UserRepo) Create(body models.User) (models.User, error) {
	result := ur.DB.Where("email = ?", body.Email).First(&body)
	if result.RowsAffected == 0 {
		ur.DB.Create(&body)
		return body, nil
	} else {
		return body, errors.New("user already exists")
	}
}

func (ur UserRepo) GetList() []models.User {
	var users []models.User
	ur.DB.Find(&users)
	return users
}

func (ur UserRepo) GetOne(id uint) (models.User, error) {
	var user models.User
	result := ur.DB.Where("id = ?", id).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("user not found")
	} else {
		return user, nil
	}
}

func (ur UserRepo) Update(u uint, body models.User) (models.User, error) {
	//TODO implement me
	panic("implement me")
}

func (ur UserRepo) Delete(id uint) error {
	result := ur.DB.Where("id = ?", id).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	} else {
		return nil
	}
}
