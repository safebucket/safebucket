package services

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	dbsql "github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const serviceAccountEmailDomain = "service-account.local"

var serviceAccountSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type ServiceAccountService struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

func (s ServiceAccountService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		Get("/", handlers.GetListHandler(s.ListServiceAccounts))
	r.With(m.AuthorizeRole(models.RoleAdmin)).
		With(m.Validate[models.ServiceAccountCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateServiceAccount))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.AuthorizeRole(models.RoleAdmin)).
			With(m.Validate[models.ServiceAccountUpdateBody]).
			Patch("/", handlers.BodyHandler(s.UpdateServiceAccount))
		r.With(m.AuthorizeRole(models.RoleAdmin)).
			Delete("/", handlers.DeleteHandler(s.DeleteServiceAccount))

		r.Mount("/tokens", TokenService{
			DB:             s.DB,
			ActivityLogger: s.ActivityLogger,
		}.AdminRoutes())
	})

	return r
}

func (s ServiceAccountService) ListServiceAccounts(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
) []models.ServiceAccountResponse {
	var accounts []models.User
	if err := s.DB.Scopes(dbsql.OnlyServiceAccounts).
		Order("created_at DESC").Find(&accounts).Error; err != nil {
		logger.Error("Failed to list service accounts", zap.Error(err))
		return []models.ServiceAccountResponse{}
	}

	result := make([]models.ServiceAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, models.NewServiceAccountResponse(account))
	}
	return result
}

func (s ServiceAccountService) CreateServiceAccount(
	logger *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
	body models.ServiceAccountCreateBody,
) (models.ServiceAccountResponse, error) {
	if body.Role == models.RoleAdmin && !body.IsAdmin {
		return models.ServiceAccountResponse{}, apierrors.New(
			http.StatusBadRequest, apierrors.CodeServiceAccountAdminRequired)
	}

	slug := serviceAccountSlugPattern.ReplaceAllString(strings.ToLower(body.Name), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "sa"
	}
	suffix := strings.Split(uuid.NewString(), "-")[0]
	email := fmt.Sprintf("%s-%s@%s", slug, suffix, serviceAccountEmailDomain)

	account := models.User{
		FirstName:    body.Name,
		Email:        email,
		ProviderType: models.ServiceAccountProviderType,
		ProviderKey:  string(models.ServiceAccountProviderType),
		Role:         body.Role,
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&account).Error; createErr != nil {
			return createErr
		}

		message := activity.ServiceAccountCreated
		action := models.Activity{
			Message: message,
			Object:  account.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				ObjectType: rbac.ResourceServiceAccount.String(),
				UserID:     claims.UserID.String(),
			}),
		}
		return s.ActivityLogger.Send(action)
	})
	if err != nil {
		logger.Error("Failed to create service account", zap.Error(err))
		return models.ServiceAccountResponse{}, apierrors.New(
			http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	return models.NewServiceAccountResponse(account), nil
}

func (s ServiceAccountService) UpdateServiceAccount(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.ServiceAccountUpdateBody,
) error {
	var account models.User
	err := s.DB.Scopes(dbsql.OnlyServiceAccounts).Where("id = ?", ids[0]).Take(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeServiceAccountNotFound)
		}
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	updates := map[string]any{}
	if body.Name != nil {
		updates["first_name"] = *body.Name
	}
	if body.Role != nil {
		if *body.Role == models.RoleAdmin && !body.IsAdmin {
			return apierrors.New(http.StatusBadRequest, apierrors.CodeServiceAccountAdminRequired)
		}
		updates["role"] = *body.Role
	}

	if len(updates) == 0 {
		return nil
	}

	if updErr := s.DB.Model(&models.User{}).Where("id = ?", account.ID).
		Updates(updates).Error; updErr != nil {
		logger.Error("Failed to update service account", zap.Error(updErr))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	return nil
}

func (s ServiceAccountService) DeleteServiceAccount(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
) error {
	var account models.User
	err := s.DB.Scopes(dbsql.OnlyServiceAccounts).Where("id = ?", ids[0]).Take(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeServiceAccountNotFound)
		}
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if delErr := tx.Unscoped().Where("user_id = ?", account.ID).
			Delete(&models.APIToken{}).Error; delErr != nil {
			return delErr
		}
		if delErr := tx.Where("user_id = ?", account.ID).
			Delete(&models.Membership{}).Error; delErr != nil {
			return delErr
		}
		if delErr := tx.Where("id = ?", account.ID).Delete(&models.User{}).Error; delErr != nil {
			return delErr
		}

		action := models.Activity{
			Message: activity.ServiceAccountDeleted,
			Object:  account.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				ObjectType: rbac.ResourceServiceAccount.String(),
				UserID:     claims.UserID.String(),
			}),
		}
		return s.ActivityLogger.Send(action)
	})
	if err != nil {
		logger.Error("Failed to delete service account", zap.Error(err))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	return nil
}
