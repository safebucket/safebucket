package services

import (
	customerrors "api/internal/errors"
	"api/internal/handlers"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/sql"
	"errors"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func (s UserService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		Get("/", handlers.GetListHandler(s.GetUserList))

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		With(m.Validate[models.UserCreateBody]).Post("/", handlers.CreateHandler(s.CreateUser))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.AuthorizeSelfOrAdmin(0)).
			Get("/", handlers.GetOneHandler(s.GetUser))

		r.With(m.AuthorizeSelfOrAdmin(0)).
			With(m.Validate[models.UserUpdateBody]).Patch("/", handlers.UpdateHandler(s.UpdateUser))

		r.With(m.AuthorizeRole(models.RoleAdmin)).
			Delete("/", handlers.DeleteHandler(s.DeleteUser))
	})
	return r
}

func (s UserService) CreateUser(logger *zap.Logger, _ models.UserClaims, _ uuid.UUIDs, body models.UserCreateBody) (models.User, error) {
	newUser := models.User{
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		Email:        body.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}

	result := s.DB.Where("email = ?", newUser.Email).Find(&newUser)
	if result.RowsAffected == 0 {
		hash, err := h.CreateHash(body.Password)
		if err != nil {
			return models.User{}, errors.New("can not create hash password")
		}
		newUser.HashedPassword = hash

		err = sql.CreateUserWithInvites(logger, s.DB, &newUser)
		if err != nil {
			return models.User{}, customerrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		return newUser, nil
	} else {
		return models.User{}, errors.New("user already exists, try to reset your password")
	}
}

func (s UserService) GetUserList(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs) []models.User {
	var users []models.User
	s.DB.Find(&users)
	return users
}

func (s UserService) GetUser(_ *zap.Logger, _ models.UserClaims, ids uuid.UUIDs) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", ids[0]).First(&user)
	if result.RowsAffected == 0 {
		return user, errors.New("USER_NOT_FOUND")
	} else {
		return user, nil
	}
}

func (s UserService) UpdateUser(logger *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.UserUpdateBody) (models.User, error) {
	user := models.User{ID: ids[0]}

	updatedUser := models.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
	}

	if body.OldPassword != "" && body.NewPassword != "" {
		user.ProviderType = models.LocalProviderType
		user.ProviderKey = string(models.LocalProviderType)

		result := s.DB.Where(user, "id", "provider_type", "provider_key").Find(&user)
		if result.RowsAffected == 0 {
			return user, errors.New("USER_NOT_FOUND")
		}

		match, err := argon2id.ComparePasswordAndHash(body.OldPassword, user.HashedPassword)
		if err != nil {
			return models.User{}, errors.New("INTERNAL_SERVER_ERROR")
		}
		if !match {
			return models.User{}, errors.New("INCORRECT_PASSWORD")
		}

		hash, err := h.CreateHash(body.NewPassword)
		if err != nil {
			return models.User{}, errors.New("INTERNAL_SERVER_ERROR")
		}

		// The password can be updated after passing all the checks
		updatedUser.HashedPassword = hash
	} else {
		result := s.DB.Where(user, "id").Find(&user)
		if result.RowsAffected == 0 {
			return user, errors.New("USER_NOT_FOUND")
		}
	}

	result := s.DB.Model(&user).Updates(updatedUser)
	if result.RowsAffected == 0 {
		return models.User{}, errors.New("USER_NOT_FOUND")
	} else {
		return models.User{}, nil
	}
}

func (s UserService) DeleteUser(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	userId := ids[0]

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ?", userId).Delete(&models.User{})
		if result.RowsAffected == 0 {
			return errors.New("USER_NOT_FOUND")
		}

		if result.Error != nil {
			logger.Error("Failed to delete user", zap.Error(result.Error), zap.String("user_id", userId.String()))
			return result.Error
		}

		// Delete user-created invites
		result = tx.Where("created_by = ?", userId.String()).Delete(&models.Invite{})
		if result.Error != nil {
			logger.Error("Failed to delete user-created invites", zap.Error(result.Error), zap.String("user_id", userId.String()))
			return result.Error
		}

		// Note: User's memberships will be cascade deleted by the foreign key constraint

		return nil
	})

	if err != nil {
		return customerrors.ErrorInternalServer
	}

	logger.Info("User successfully deleted", zap.String("user_id", userId.String()), zap.String("email", user.Email))
	return nil
}
