package services

import (
	"api/internal/handlers"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/roles"
	"api/internal/sql"
	"errors"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	DB       *gorm.DB
	Enforcer *casbin.Enforcer
}

func (s UserService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceUser, rbac.ActionList, -1)).
		Get("/", handlers.GetListHandler(s.GetUserList))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceUser, rbac.ActionCreate, -1)).
		With(m.Validate[models.UserCreateBody]).Post("/", handlers.CreateHandler(s.CreateUser))

	r.Route("/{id0}", func(r chi.Router) {

		r.With(m.Authorize(s.Enforcer, rbac.ResourceUser, rbac.ActionRead, 0)).
			Get("/", handlers.GetOneHandler(s.GetUser))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceUser, rbac.ActionUpdate, 0)).
			With(m.Validate[models.UserUpdateBody]).Patch("/", handlers.UpdateHandler(s.UpdateUser))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceUser, rbac.ActionDelete, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(_ models.UserClaims, _ uuid.UUIDs, body models.UserCreateBody) (models.User, error) {
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

		err = sql.CreateUserWithRole(s.DB, s.Enforcer, &newUser, roles.AddUserToRoleUser)
		if err != nil {
			return models.User{}, err
		}

		return newUser, nil
	} else {
		return models.User{}, errors.New("user already exists, try to reset your password")
	}
}

func (s UserService) GetUserList(_ models.UserClaims, _ uuid.UUIDs) []models.User {
	var users []models.User
	s.DB.Find(&users)
	return users
}

func (s UserService) GetUser(_ models.UserClaims, ids uuid.UUIDs) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", ids[0]).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("USER_NOT_FOUND")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(_ models.UserClaims, ids uuid.UUIDs, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: ids[0]}

	newUser := models.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
	}

	if body.Password != "" {
		hash, err := h.CreateHash(body.Password)
		if err != nil {
			return user, errors.New("can not create hash password")
		}
		newUser.HashedPassword = hash
	}
	result := s.DB.Model(&user).Updates(newUser)
	if result.RowsAffected == 0 {
		return user, errors.New("USER_NOT_FOUND")
	} else {
		return user, nil
	}
}

func (s UserService) DeleteUser(_ models.UserClaims, ids uuid.UUIDs) error {
	result := s.DB.Where("id = ?", ids[0]).Delete(&models.User{})
	if result.RowsAffected == 0 {
		return errors.New("USER_NOT_FOUND")
	} else {
		return nil
	}
}
