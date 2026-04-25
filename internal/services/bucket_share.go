package services

import (
	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketShareService struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

func (s BucketShareService) Routes() chi.Router {
	r := chi.NewRouter()
	authorize := m.AuthorizeGroup(s.DB, models.GroupOwner, 0)

	r.With(authorize).Get("/", handlers.GetListHandler(s.ListShares))
	r.With(authorize, m.Validate[models.ShareCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateShare))
	r.With(authorize).Delete("/{id1}", handlers.DeleteHandler(s.DeleteShare))

	return r
}

func (s BucketShareService) CreateShare(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.ShareCreateBody,
) (models.Share, error) {
	bucketID := ids[0]

	var bucket models.Bucket
	if s.DB.Where("id = ?", bucketID).Find(&bucket).RowsAffected == 0 {
		return models.Share{}, apierrors.NewAPIError(404, "NOT_FOUND")
	}

	if body.Type == models.ShareTypeFolder {
		var folder models.Folder
		if s.DB.Where("id = ? AND bucket_id = ? AND status = ?", body.FolderID, bucketID, models.FolderStatusCreated).
			Find(&folder).
			RowsAffected == 0 {
			return models.Share{}, apierrors.NewAPIError(400, "FOLDER_NOT_FOUND")
		}
	}

	if body.Type == models.ShareTypeFiles {
		var count int64
		s.DB.Model(&models.File{}).
			Where("id IN ? AND bucket_id = ? AND status = ?", body.FileIDs, bucketID, models.FileStatusUploaded).
			Count(&count)
		if count != int64(len(body.FileIDs)) {
			return models.Share{}, apierrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
	}

	var hashedPassword string
	if body.Password != "" {
		hash, err := h.CreateHash(body.Password)
		if err != nil {
			logger.Error("Failed to hash share password", zap.Error(err))
			return models.Share{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
		hashedPassword = hash
	}

	share := &models.Share{
		Name:           body.Name,
		BucketID:       bucketID,
		FolderID:       body.FolderID,
		ExpiresAt:      body.ExpiresAt,
		MaxViews:       body.MaxViews,
		HashedPassword: hashedPassword,
		Type:           body.Type,
		AllowUpload:    body.AllowUpload,
		MaxUploads:     body.MaxUploads,
		MaxUploadSize:  body.MaxUploadSize,
		CreatedBy:      user.UserID,
	}

	txErr := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(share).Error; err != nil {
			logger.Error("Failed to create share", zap.Error(err))
			return err
		}

		if body.Type == models.ShareTypeFiles {
			shareFiles := make([]models.ShareFile, len(body.FileIDs))
			for i, fileID := range body.FileIDs {
				shareFiles[i] = models.ShareFile{
					ShareID: share.ID,
					FileID:  fileID,
				}
			}
			if err := tx.Create(&shareFiles).Error; err != nil {
				logger.Error("Failed to create share files", zap.Error(err))
				return err
			}
			share.Files = shareFiles
		}

		if err := s.ActivityLogger.Send(models.Activity{
			Message: activity.ShareCreated,
			Object:  share.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				ObjectType: rbac.ResourceShare.String(),
				BucketID:   bucketID.String(),
				ShareID:    share.ID.String(),
				UserID:     user.UserID.String(),
			}),
		}); err != nil {
			logger.Error("Failed to log share create activity", zap.Error(err))
		}

		return nil
	})

	if txErr != nil {
		return models.Share{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	share.PasswordProtected = hashedPassword != ""

	return *share, nil
}

func (s BucketShareService) ListShares(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.Share {
	bucketID := ids[0]

	var shares []models.Share
	err := s.DB.Where("bucket_id = ?", bucketID).
		Preload("Files.File").
		Order("created_at DESC").
		Find(&shares).Error

	if err != nil {
		logger.Error("Failed to list shares", zap.Error(err))
		return []models.Share{}
	}

	for i := range shares {
		shares[i].PasswordProtected = shares[i].HashedPassword != ""
	}

	return shares
}

func (s BucketShareService) DeleteShare(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, shareID := ids[0], ids[1]

	return s.DB.Transaction(func(tx *gorm.DB) error {
		var share models.Share
		result := tx.Where("id = ? AND bucket_id = ?", shareID, bucketID).First(&share)
		if result.Error != nil {
			return apierrors.NewAPIError(404, "NOT_FOUND")
		}

		if err := tx.Delete(&share).Error; err != nil {
			logger.Error("Failed to delete share", zap.Error(err))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		err := s.ActivityLogger.Send(models.Activity{
			Message: activity.ShareDeleted,
			Object:  share.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				ObjectType: rbac.ResourceShare.String(),
				BucketID:   bucketID.String(),
				ShareID:    shareID.String(),
				UserID:     user.UserID.String(),
			}),
		})

		if err != nil {
			logger.Error("Failed to log share delete activity", zap.Error(err))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		return nil
	})
}
