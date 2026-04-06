package services

import (
	"path"
	"path/filepath"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PublicShareService struct {
	DB             *gorm.DB
	Storage        storage.IStorage
	ActivityLogger activity.IActivityLogger
	JWTSecret      string
}

func (s PublicShareService) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{id0}", func(r chi.Router) {
		r.Use(m.ValidateShareAccess(s.DB))

		r.With(m.Validate[models.ShareAuthBody]).
			Post("/auth", handlers.ShareAuthHandler(s.AuthenticateShare))

		r.Group(func(r chi.Router) {
			r.Use(m.ValidateShareToken(s.JWTSecret))

			r.Get("/", handlers.ShareGetOneHandler(s.ListShareItems))
			r.Get("/files/{id1}", handlers.ShareGetOneHandler(s.DownloadShareFile))
			r.With(m.Validate[models.ShareUploadBody]).
				Post("/files", handlers.ShareCreateHandler(s.UploadShareFile))
			r.Patch("/files/{id1}", handlers.ShareActionHandler(s.ConfirmShareUpload))
		})
	})

	return r
}

func (s PublicShareService) AuthenticateShare(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
	body models.ShareAuthBody,
) (models.ShareAuthResponse, error) {
	if share.HashedPassword == "" {
		return models.ShareAuthResponse{}, apierrors.NewAPIError(400, "SHARE_NOT_PASSWORD_PROTECTED")
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, share.HashedPassword)
	if err != nil || !match {
		return models.ShareAuthResponse{}, apierrors.NewAPIError(401, "SHARE_PASSWORD_INVALID")
	}

	token, err := h.NewShareAccessToken(s.JWTSecret, share.ID)
	if err != nil {
		logger.Error("Failed to create share access token", zap.Error(err))
		return models.ShareAuthResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.ShareAuthResponse{Token: token}, nil
}

func (s PublicShareService) ListShareItems(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
) (models.PublicShareResponse, error) {
	result := s.DB.Model(&models.Share{}).
		Where("id = ? AND (max_views IS NULL OR current_views < max_views)", share.ID).
		UpdateColumn("current_views", gorm.Expr("current_views + 1"))
	if result.Error != nil {
		logger.Error("Failed to increment share views", zap.Error(result.Error))
		return models.PublicShareResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	if result.RowsAffected == 0 {
		return models.PublicShareResponse{}, apierrors.NewAPIError(403, "SHARE_MAX_VIEWS_REACHED")
	}

	response := models.PublicShareResponse{
		ID:            share.ID,
		Name:          share.Name,
		Type:          share.Type,
		AllowUpload:   share.AllowUpload,
		MaxUploadSize: share.MaxUploadSize,
		Files:         []models.File{},
		Folders:       []models.Folder{},
	}

	now := time.Now()

	switch share.Type {
	case models.ShareTypeFiles:
		var files []models.File
		s.DB.Joins("JOIN share_files ON share_files.file_id = files.id").
			Where("share_files.share_id = ?", share.ID).
			Where("files.status = ?", models.FileStatusUploaded).
			Where("files.expires_at IS NULL OR files.expires_at > ?", now).
			Find(&files)
		response.Files = files

	case models.ShareTypeFolder:
		var files []models.File
		s.DB.Where(
			"bucket_id = ? AND folder_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)",
			share.BucketID, share.FolderID, models.FileStatusUploaded, now,
		).Find(&files)
		response.Files = files

		var folders []models.Folder
		s.DB.Where("bucket_id = ? AND folder_id = ? AND status = ?", share.BucketID, share.FolderID, models.FolderStatusCreated).
			Find(&folders)
		response.Folders = folders

	case models.ShareTypeBucket:
		var files []models.File
		s.DB.Where(
			"bucket_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)",
			share.BucketID, models.FileStatusUploaded, now,
		).Find(&files)
		response.Files = files

		var folders []models.Folder
		s.DB.Where("bucket_id = ? AND status = ?", share.BucketID, models.FolderStatusCreated).Find(&folders)
		response.Folders = folders
	}

	return response, nil
}

func (s PublicShareService) DownloadShareFile(
	logger *zap.Logger,
	share models.Share,
	ids uuid.UUIDs,
) (models.FileTransferResponse, error) {
	fileID := ids[1]

	file, err := h.GetShareFile(s.DB, share, fileID)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	url, err := s.Storage.PresignedGetObject(
		path.Join("buckets", share.BucketID.String(), file.ID.String()),
	)
	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		return models.FileTransferResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	if activityErr := s.ActivityLogger.Send(models.Activity{
		Message: activity.ShareFileDownloaded,
		Object:  file.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDownload.String(),
			"object_type": rbac.ResourceFile.String(),
			"bucket_id":   share.BucketID.String(),
			"file_id":     fileID.String(),
			"share_id":    share.ID.String(),
		}),
	}); activityErr != nil {
		logger.Error("Failed to log share download activity", zap.Error(activityErr))
		return models.FileTransferResponse{}, activityErr
	}

	return models.FileTransferResponse{
		ID:  file.ID.String(),
		URL: url,
	}, nil
}

func (s PublicShareService) UploadShareFile(
	logger *zap.Logger,
	share models.Share,
	_ uuid.UUIDs,
	body models.ShareUploadBody,
) (models.FileTransferResponse, error) {
	if !share.AllowUpload {
		return models.FileTransferResponse{}, apierrors.NewAPIError(403, "SHARE_UPLOAD_NOT_ALLOWED")
	}

	if share.MaxUploadSize != nil && body.Size > *share.MaxUploadSize {
		return models.FileTransferResponse{}, apierrors.NewAPIError(400, "SHARE_UPLOAD_SIZE_EXCEEDED")
	}

	if share.MaxUploads != nil && share.CurrentUploads >= *share.MaxUploads {
		return models.FileTransferResponse{}, apierrors.NewAPIError(403, "MAX_UPLOADS_REACHED")
	}

	var folderID *uuid.UUID
	switch share.Type {
	case models.ShareTypeFiles:
		folderID = nil
	case models.ShareTypeFolder:
		folderID = share.FolderID
	case models.ShareTypeBucket:
		if body.FolderID != nil {
			folderID = body.FolderID
			var folder models.Folder
			if s.DB.Where("id = ? AND bucket_id = ?", folderID, share.BucketID).
				Find(&folder).RowsAffected == 0 {
				return models.FileTransferResponse{}, apierrors.NewAPIError(404, "FOLDER_NOT_FOUND")
			}
		}
	}

	var existingFile models.File
	query := s.DB.Where("bucket_id = ? AND name = ?", share.BucketID, body.Name)
	if folderID != nil {
		query = query.Where("folder_id = ?", folderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}

	if query.Find(&existingFile).RowsAffected > 0 {
		return models.FileTransferResponse{}, apierrors.NewAPIError(409, "FILE_ALREADY_EXISTS")
	}

	extension := filepath.Ext(body.Name)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	file := &models.File{
		Status:    models.FileStatusUploading,
		Name:      body.Name,
		Extension: extension,
		BucketID:  share.BucketID,
		FolderID:  folderID,
		Size:      int(body.Size),
	}

	var url string
	var formData map[string]string
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if txErr := tx.Create(file).Error; txErr != nil {
			return txErr
		}

		var presignErr error
		url, formData, presignErr = s.Storage.PresignedPostPolicy(
			path.Join("buckets", share.BucketID.String(), file.ID.String()),
			int(body.Size),
			map[string]string{
				"bucket_id": share.BucketID.String(),
				"file_id":   file.ID.String(),
				"share_id":  share.ID.String(),
			},
		)
		if presignErr != nil {
			logger.Error("Generate presigned URL failed", zap.Error(presignErr))
			return presignErr
		}

		uploadResult := tx.Model(&models.Share{}).
			Where("id = ? AND (max_uploads IS NULL OR current_uploads < max_uploads)", share.ID).
			UpdateColumn("current_uploads", gorm.Expr("current_uploads + 1"))
		if uploadResult.Error != nil {
			return uploadResult.Error
		}
		if uploadResult.RowsAffected == 0 {
			return apierrors.NewAPIError(403, "SHARE_MAX_UPLOADS_REACHED")
		}

		if activityErr := s.ActivityLogger.Send(models.Activity{
			Message: activity.ShareFileUploaded,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionCreate.String(),
				"object_type": rbac.ResourceFile.String(),
				"bucket_id":   share.BucketID.String(),
				"file_id":     file.ID.String(),
				"share_id":    share.ID.String(),
			}),
		}); activityErr != nil {
			logger.Warn("Failed to log share upload activity", zap.Error(activityErr))
		}

		return nil
	})

	if err != nil {
		return models.FileTransferResponse{}, err
	}

	return models.FileTransferResponse{
		ID:   file.ID.String(),
		URL:  url,
		Body: formData,
	}, nil
}

func (s PublicShareService) ConfirmShareUpload(
	logger *zap.Logger,
	share models.Share,
	ids uuid.UUIDs,
) error {
	fileID := ids[1]

	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", fileID, share.BucketID).
			Find(&file)

		if result.Error != nil {
			logger.Error("Find file failed", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
		}

		if file.Status != models.FileStatusUploading {
			return apierrors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
		}

		if !h.IsFileInShare(tx, share, fileID, file) {
			return apierrors.NewAPIError(403, "SHARE_FILE_NOT_IN_SHARE")
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())
		if _, statErr := s.Storage.StatObject(objectPath); statErr != nil {
			logger.Error("File not found in storage",
				zap.Error(statErr),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()))
			return apierrors.NewAPIError(404, "FILE_NOT_IN_STORAGE")
		}

		if txErr := tx.Model(&file).Update("status", models.FileStatusUploaded).Error; txErr != nil {
			logger.Error("Failed to update file status", zap.Error(txErr))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		return nil
	})
}
