package services

import (
	"errors"
	"net/http"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService struct {
	DB                 *gorm.DB
	Cache              cache.ICache
	AuthConfig         models.AuthConfig
	Publisher          messaging.IPublisher
	Notifier           notifier.INotifier
	ActivityLogger     activity.IActivityLogger
	RefreshTokenExpiry int
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
			With(m.Validate[models.UserUpdateBody]).Patch("/", handlers.BodyHandler(s.UpdateUser))

		r.With(m.AuthorizeRole(models.RoleAdmin)).
			Delete("/", handlers.DeleteHandler(s.DeleteUser))

		r.With(m.AuthorizeSelfOrAdmin(0)).
			Get("/stats", handlers.GetOneHandler(s.GetUserStats))

		r.Mount("/sessions", SessionService{
			Cache:              s.Cache,
			RefreshTokenExpiry: s.RefreshTokenExpiry,
			ActivityLogger:     s.ActivityLogger,
		}.Routes())
	})
	return r
}

func (s UserService) CreateUser(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.UserCreateBody,
) (models.User, error) {
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
			return models.User{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		newUser.HashedPassword = hash

		err = sql.CreateUserWithInvites(logger, s.DB, &newUser)
		if err != nil {
			return models.User{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		return newUser, nil
	}
	return models.User{}, apierrors.New(http.StatusConflict, apierrors.CodeUserAlreadyExists)
}

func (s UserService) GetUserList(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs) []models.User {
	var users []models.User
	s.DB.Find(&users)
	return users
}

func (s UserService) GetUser(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (models.User, error) {
	var user models.User
	result := s.DB.Where("id = ?", ids[0]).First(&user)
	if result.RowsAffected == 0 {
		return user, apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
	}
	return user, nil
}

func (s UserService) UpdateUser(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.UserUpdateBody,
) error {
	user := models.User{ID: ids[0]}

	updatedUser := models.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
	}

	if body.OldPassword != "" && body.NewPassword != "" {
		loaded, err := sql.GetUserByID(s.DB, ids[0])
		if err != nil {
			return err
		}
		if loaded.ProviderType != models.LocalProviderType {
			return apierrors.New(http.StatusForbidden, apierrors.CodePasswordChangeNotAllowed)
		}
		user = loaded

		match, err := argon2id.ComparePasswordAndHash(body.OldPassword, user.HashedPassword)
		if err != nil {
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		if !match {
			return apierrors.New(http.StatusUnauthorized, apierrors.CodeIncorrectPassword)
		}

		hash, err := h.CreateHash(body.NewPassword)
		if err != nil {
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		updatedUser.HashedPassword = hash
	} else {
		result := s.DB.Where(user, "id").Find(&user)
		if result.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
		}
	}

	result := s.DB.Model(&user).Updates(updatedUser)
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
	}
	return nil
}

func (s UserService) DeleteUser(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	userID := ids[0]

	if err := cache.RevokeAllSessions(s.Cache, userID.String()); err != nil {
		logger.Error("Failed to revoke user sessions", zap.Error(err))
	}

	var deletedUser models.User
	if err := s.DB.Where("id = ?", userID).First(&deletedUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
		}
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ?", userID).Delete(&models.User{})
		if result.Error != nil {
			logger.Error(
				"Failed to delete user",
				zap.Error(result.Error),
				zap.String("user_id", userID.String()),
			)
			return result.Error
		}

		result = tx.Where("user_id = ?", userID).Delete(&models.Membership{})
		if result.Error != nil {
			logger.Error(
				"Failed to delete user memberships",
				zap.Error(result.Error),
				zap.String("user_id", userID.String()),
			)
			return result.Error
		}

		result = tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Challenge{})
		if result.Error != nil {
			logger.Error(
				"Failed to delete user challenges",
				zap.Error(result.Error),
				zap.String("user_id", userID.String()),
			)
			return result.Error
		}

		result = tx.Unscoped().Where("user_id = ?", userID).Delete(&models.MFADevice{})
		if result.Error != nil {
			logger.Error(
				"Failed to delete user MFA devices",
				zap.Error(result.Error),
				zap.String("user_id", userID.String()),
			)
			return result.Error
		}

		result = tx.Where("created_by = ?", userID.String()).Delete(&models.Invite{})
		if result.Error != nil {
			logger.Error(
				"Failed to delete user-created invites",
				zap.Error(result.Error),
				zap.String("user_id", userID.String()),
			)
			return result.Error
		}

		action := models.Activity{
			Message: activity.UserDeleted,
			Object:  deletedUser.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				ObjectType: rbac.ResourceUser.String(),
				UserID:     user.UserID.String(),
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log user deletion activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})

	if err != nil {
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	logger.Info(
		"User successfully deleted",
		zap.String("user_id", userID.String()),
		zap.String("email", user.Email),
	)
	return nil
}

func (s UserService) GetUserStats(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (models.UserStatsResponse, error) {
	userID := ids[0]

	var user models.User
	result := s.DB.Where("id = ?", userID).First(&user)
	if result.RowsAffected == 0 {
		return models.UserStatsResponse{}, apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
	}

	var totalBuckets int64
	s.DB.Model(&models.Membership{}).Where("user_id = ?", userID).Count(&totalBuckets)

	var totalFiles int64
	s.DB.Model(&models.File{}).
		Joins("INNER JOIN memberships ON files.bucket_id = memberships.bucket_id").
		Where("memberships.user_id = ?", userID).
		Count(&totalFiles)

	return models.UserStatsResponse{
		TotalFiles:   int(totalFiles),
		TotalBuckets: int(totalBuckets),
	}, nil
}
