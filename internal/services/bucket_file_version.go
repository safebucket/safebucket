package services

import (
	"net/http"
	"strconv"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketFileVersionService struct {
	BucketFileService
}

func (s BucketFileVersionService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
		Get("/", handlers.GetListHandler(s.ListFileVersions))

	r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
		With(m.ValidateQuery[models.FileDownloadQuery]).
		Get("/{id2}/url", handlers.GetOneWithQueryHandler(s.DownloadFileVersion))

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		Post("/{id2}/confirm", handlers.ActionHandler(s.ConfirmFileVersion))

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		Post("/{id2}/restore", handlers.ActionHandler(s.RestoreFileVersion))

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		Delete("/{id2}", handlers.DeleteHandler(s.DeleteFileVersion))

	return r
}

func (s BucketFileVersionService) ConfirmFileVersion(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) error {
	completed, err := s.Versions.Complete(logger, ids[0], ids[1], ids[2], nil, false)
	if err == nil && completed.Changed {
		events.NotifyFileVersionUploaded(logger, s.DB, completed, nil)
	}
	return err
}

func (s BucketFileVersionService) ListFileVersions(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.FileVersionResponse {
	file, err := sql.GetFileByID(s.DB, ids[0], ids[1])
	if err != nil {
		logger.Warn("Failed to load file for version listing", zap.Error(err))
		return []models.FileVersionResponse{}
	}

	var versions []models.FileVersion
	if listErr := s.DB.Where("file_id = ? AND status = ?", file.ID, models.FileStatusUploaded).
		Preload("Uploader").
		Order("version DESC").
		Find(&versions).Error; listErr != nil {
		logger.Error("Failed to list file versions", zap.Error(listErr))
		return []models.FileVersionResponse{}
	}

	response := make([]models.FileVersionResponse, 0, len(versions))
	for _, version := range versions {
		var uploader *models.UserInfo
		if version.Uploader != nil {
			info := version.Uploader.ToActivity()
			uploader = &info
		}
		response = append(response, models.FileVersionResponse{
			ID:         version.ID,
			Version:    version.Version,
			Size:       version.Size,
			Status:     version.Status,
			UploadedBy: uploader,
			IsCurrent:  file.CurrentVersionID != nil && *file.CurrentVersionID == version.ID,
			CreatedAt:  version.CreatedAt,
		})
	}

	return response
}

func (s BucketFileVersionService) RestoreFileVersion(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		file, err := sql.LockFile(tx, ids[0], ids[1])
		if err != nil {
			return err
		}
		if err = sql.CheckFileUploadable(file); err != nil {
			return err
		}

		version, err := sql.GetFileVersionByID(tx, file.ID, ids[2])
		if err != nil {
			return err
		}
		isCurrent := file.CurrentVersionID != nil && *file.CurrentVersionID == version.ID
		if version.Status != models.FileStatusUploaded || isCurrent {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotRestorable)
		}

		if updateErr := tx.Model(&models.File{}).
			Where("id = ?", file.ID).
			Update("current_version_id", version.ID).Error; updateErr != nil {
			logger.Error("Failed to repoint file to restored version", zap.Error(updateErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if logErr := outbox.AddActivities(tx, models.Activity{
			Message: activity.FileVersionRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
				Version:    strconv.Itoa(version.Version),
			}),
		}); logErr != nil {
			logger.Error("Failed to log version restore activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})
}

func (s BucketFileVersionService) DeleteFileVersion(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	_, _, err := s.Versions.Delete(logger, ids[0], ids[1], ids[2], nil,
		func(tx *gorm.DB, file models.File, version models.FileVersion) error {
			if version.Status != models.FileStatusUploaded {
				return nil
			}
			return outbox.AddActivities(tx, models.Activity{
				Message: activity.FileVersionDeleted,
				Object:  file.ToActivity(),
				Filter: activity.NewLogFilter(models.ActivityFields{
					Action: rbac.ActionDelete.String(), BucketID: file.BucketID.String(),
					FileID: file.ID.String(), ObjectType: rbac.ResourceFile.String(),
					UserID: user.UserID.String(), Version: strconv.Itoa(version.Version),
				}),
			})
		})
	return err
}

func (s BucketFileVersionService) DownloadFileVersion(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	query models.FileDownloadQuery,
) (models.FileDownloadResponse, error) {
	file, err := s.getDownloadableFile(ids[0], ids[1])
	if err != nil {
		return models.FileDownloadResponse{}, err
	}

	version, err := sql.GetFileVersionByID(s.DB, file.ID, ids[2])
	if err != nil {
		return models.FileDownloadResponse{}, err
	}
	if version.Status != models.FileStatusUploaded {
		return models.FileDownloadResponse{}, apierrors.New(
			http.StatusNotFound,
			apierrors.CodeFileVersionNotFound,
		)
	}

	return s.downloadVersion(logger, user, file, version.ID, version.Version, query.Context)
}
