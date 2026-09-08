package services

import (
	"errors"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	"github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	dbsql "github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TokenService struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

func (s TokenService) Routes() chi.Router {
	return s.routes(m.AuthorizeSelfOrAdmin(0))
}

func (s TokenService) AdminRoutes() chi.Router {
	return s.routes(m.AuthorizeRole(models.RoleAdmin))
}

func (s TokenService) routes(authz func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.With(authz).Get("/", handlers.GetListHandler(s.ListTokens))
	r.With(authz).With(m.Validate[models.TokenCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateToken))
	r.Route("/{id1}", func(r chi.Router) {
		r.With(authz).Delete("/", handlers.DeleteHandler(s.RevokeToken))
	})

	return r
}

func (s TokenService) ListTokens(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.APIToken {
	var tokens []models.APIToken
	err := s.DB.Where("user_id = ? AND deleted_at IS NULL", ids[0]).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		logger.Error("Failed to list api tokens", zap.Error(err))
		return []models.APIToken{}
	}
	return tokens
}

func (s TokenService) CreateToken(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.TokenCreateBody,
) (models.TokenCreateResponse, error) {
	if claims.AudienceString() == configuration.AudienceAPIToken {
		return models.TokenCreateResponse{}, apierrors.New(
			http.StatusForbidden,
			apierrors.CodeTokenActionDenied,
		)
	}

	targetUserID := ids[0]
	owner, err := dbsql.GetUserByID(s.DB, targetUserID)
	if err != nil {
		return models.TokenCreateResponse{}, err
	}

	plaintext, hash, err := helpers.GenerateAPIToken(owner.ProviderType)
	if err != nil {
		return models.TokenCreateResponse{}, apierrors.New(
			http.StatusInternalServerError, apierrors.CodeTokenCreateFailed)
	}

	createdBy := claims.UserID
	token := models.APIToken{
		TokenHash: hash,
		UserID:    targetUserID,
		Name:      body.Name,
		ExpiresAt: time.Now().Add(time.Duration(body.ExpiryDays) * 24 * time.Hour),
		CreatedBy: &createdBy,
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&token).Error; createErr != nil {
			return createErr
		}
		action := models.Activity{
			Message: activity.APITokenCreated,
			Object:  token.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				ObjectType: rbac.ResourceAPIToken.String(),
				UserID:     targetUserID.String(),
			}),
		}
		return s.ActivityLogger.Send(action)
	})
	if err != nil {
		logger.Error("Failed to create api token", zap.Error(err))
		return models.TokenCreateResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeTokenCreateFailed,
		)
	}

	return models.TokenCreateResponse{APIToken: token, Secret: plaintext}, nil
}

func (s TokenService) RevokeToken(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
) error {
	targetUserID := ids[0]
	tokenID := ids[1]

	var token models.APIToken
	err := s.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", tokenID, targetUserID).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeTokenNotFound)
		}
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	if token.RevokedAt != nil {
		return nil
	}

	now := time.Now()
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if updErr := tx.Model(&models.APIToken{}).Where("id = ?", tokenID).
			Update("revoked_at", now).Error; updErr != nil {
			return updErr
		}

		action := models.Activity{
			Message: activity.APITokenRevoked,
			Object:  token.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				ObjectType: rbac.ResourceAPIToken.String(),
				UserID:     claims.UserID.String(),
			}),
		}
		return s.ActivityLogger.Send(action)
	})

	if err != nil {
		logger.Error("Failed to revoke api token", zap.Error(err))
		return apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeTokenRevokeFailed,
		)
	}

	return nil
}
