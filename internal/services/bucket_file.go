package services

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BucketFileService struct {
	DB                 *gorm.DB
	Cache              cache.ICache
	Storage            storage.IStorage
	Publisher          messaging.IPublisher
	ActivityLogger     activity.IActivityLogger
	TrashRetentionDays int
}

func (s BucketFileService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FileUploadBody]).
		Post("/files", handlers.CreateHandler(s.UploadFile))

	r.Route("/files/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FilePatchBody]).
			Patch("/", handlers.BodyHandler(s.PatchFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FileVersionRestoreBody]).
			Put("/", handlers.BodyHandler(s.RestoreFileVersion))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			With(m.ValidateQuery[models.FileDownloadQuery]).
			Get("/url", handlers.GetOneWithQueryHandler(s.DownloadFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/versions", handlers.GetOneHandler(s.ListFileVersions))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			With(m.ValidateQuery[models.FileDownloadQuery]).
			Get("/versions/{id2}/url", handlers.GetOneWithQueryHandler(s.DownloadFileVersion))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/versions/{id2}", handlers.DeleteHandler(s.DeleteFileVersion))
	})

	return r
}

func (s BucketFileService) UploadFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileUploadBody,
) (models.FileUploadResponse, error) {
	var bucket models.Bucket
	result := s.DB.Where("id = ?", ids[0]).Find(&bucket)
	if result.RowsAffected == 0 {
		return models.FileUploadResponse{}, apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
	}

	if body.FolderID != nil {
		var folder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ?", body.FolderID, bucket.ID).Find(&folder)
		if result.RowsAffected == 0 {
			return models.FileUploadResponse{}, apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
		}
	}

	var existingFile models.File
	query := s.DB.Where("bucket_id = ? AND name = ?", bucket.ID, body.Name)
	if body.FolderID != nil {
		query = query.Where("folder_id = ?", body.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFile)
	if result.RowsAffected > 0 {
		return models.FileUploadResponse{}, apierrors.New(http.StatusConflict, apierrors.CodeFileAlreadyExists)
	}

	file := &models.File{
		Status:    models.FileStatusUploading,
		Name:      body.Name,
		Extension: h.ExtensionFromName(body.Name),
		BucketID:  bucket.ID,
		FolderID:  body.FolderID,
		Size:      body.Size,
		ExpiresAt: body.ExpiresAt,
	}

	var response models.FileUploadResponse
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Create(file)
		if res.Error != nil {
			return res.Error
		}

		objectPath := path.Join("buckets", bucket.ID.String(), file.ID.String())

		presigned, presignErr := s.Storage.PresignUpload(
			objectPath,
			body.Size,
			map[string]string{
				"bucket_id": bucket.ID.String(),
				"file_id":   file.ID.String(),
				"user_id":   user.UserID.String(),
			},
		)
		if presignErr != nil {
			logger.Error("Presign upload failed", zap.Error(presignErr))
			return presignErr
		}
		response = presigned.Response

		if presigned.UploadID != "" {
			state := cache.MultipartState{UploadID: presigned.UploadID, PartSize: presigned.PartSize}
			if cacheErr := cache.SetMultipartState(s.Cache, file.ID.String(), state); cacheErr != nil {
				if abortErr := s.Storage.AbortMultipartUpload(objectPath, presigned.UploadID); abortErr != nil {
					logger.Warn("Failed to abort orphaned multipart upload", zap.Error(abortErr))
				}
				return cacheErr
			}
		}

		return nil
	})
	if err != nil {
		return models.FileUploadResponse{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeCreateFailed)
	}

	response.ID = file.ID.String()

	return response, nil
}

func (s BucketFileService) PatchFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FilePatchBody,
) error {
	bucketID, fileID := ids[0], ids[1]

	var file models.File
	result := s.DB.Unscoped().
		Where("id = ? AND bucket_id = ?", fileID, bucketID).
		Find(&file)
	if result.Error != nil {
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
	}

	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return apierrors.New(http.StatusForbidden, apierrors.CodeFileExpired)
	}

	switch body.Status {
	case string(models.FileStatusDeleted):
		return s.TrashFile(logger, user, file)
	case string(models.FileStatusUploaded):
		if file.DeletedAt.Valid {
			return s.RestoreFile(logger, user, file)
		}
		return s.HandleUploadedStatus(logger, user, file)
	default:
		return apierrors.New(http.StatusBadRequest, apierrors.CodeInvalidStatus)
	}
}

// HandleUploadedStatus confirms a file upload by transitioning from "uploading" to "uploaded".
// This is required for S3 providers that don't support bucket notifications.
// The client must call this after completing the upload via the presigned URL.
func (s BucketFileService) HandleUploadedStatus(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if file.Status != models.FileStatusUploading {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		multipart, isMultipart, cacheErr := cache.GetMultipartState(s.Cache, file.ID.String())
		if cacheErr != nil {
			logger.Error("Failed to read multipart state", zap.Error(cacheErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if isMultipart {
			if completeErr := storage.FinalizeMultipartUpload(
				s.Storage, objectPath, multipart.UploadID, multipart.PartSize, int64(file.Size),
				map[string]string{
					"bucket_id": file.BucketID.String(),
					"file_id":   file.ID.String(),
					"user_id":   user.UserID.String(),
				},
			); completeErr != nil {
				if errors.Is(completeErr, storage.ErrMultipartPartMismatch) {
					return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
				}
				logger.Error("Failed to complete multipart upload",
					zap.Error(completeErr), zap.String("path", objectPath))
				return apierrors.New(http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
			}
		}

		if _, err := s.Storage.StatObject(objectPath); err != nil {
			logger.Error("File not found in storage",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()))
			return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotInStorage)
		}

		if err := tx.Model(&file).Update("status", models.FileStatusUploaded).Error; err != nil {
			logger.Error("Failed to update file status", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if isMultipart {
			if delErr := cache.DeleteMultipartState(s.Cache, file.ID.String()); delErr != nil {
				logger.Warn("Failed to delete multipart state from cache", zap.Error(delErr))
			}
		}

		if err := s.ActivityLogger.Send(models.Activity{
			Message: activity.FileUploaded,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
		}); err != nil {
			logger.Warn("Failed to log upload activity", zap.Error(err))
		}

		var bucket models.Bucket
		if dbErr := s.DB.Where("id = ?", file.BucketID).First(&bucket).Error; dbErr == nil {
			evt := events.NewFileActivityNotification(
				s.Publisher, events.FileActivityUpload, events.FileActivitySourceUser,
				file.BucketID, bucket.Name, file.Name, user.UserID, user.Email,
			)
			evt.Trigger()
		}

		return nil
	})
}

func (s BucketFileService) DeleteFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	return s.PurgeFile(logger, user, bucketID, fileID)
}

func (s BucketFileService) getDownloadableFile(bucketID, fileID uuid.UUID) (models.File, error) {
	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return models.File{}, err
	}

	if file.DeletedAt.Valid {
		return models.File{}, apierrors.New(
			http.StatusForbidden,
			apierrors.CodeCannotDownloadTrashed,
		)
	}

	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return models.File{}, apierrors.New(
			http.StatusForbidden,
			apierrors.CodeFileExpired,
		)
	}

	return file, nil
}

func (s BucketFileService) DownloadFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	query models.FileDownloadQuery,
) (models.FileDownloadResponse, error) {
	file, err := s.getDownloadableFile(ids[0], ids[1])
	if err != nil {
		return models.FileDownloadResponse{}, err
	}

	versionID := file.ID
	if file.CurrentVersionID != nil {
		versionID = *file.CurrentVersionID
	}

	return s.downloadVersion(logger, user, file, versionID, 0, query.Context)
}

func (s BucketFileService) DownloadFileVersion(
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

func (s BucketFileService) downloadVersion(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
	versionID uuid.UUID,
	versionNumber int,
	context string,
) (models.FileDownloadResponse, error) {
	objectPath := sql.VersionObjectKey(file.BucketID, versionID)

	var inlineContentType string
	if context == "preview" {
		inlineContentType = h.PreviewMimeFromExtension(file.Extension)
	}

	url, err := s.Storage.PresignedGetObject(objectPath, storage.GetObjectOptions{
		InlineContentType: inlineContentType,
		DownloadFilename:  file.Name,
	})
	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		return models.FileDownloadResponse{}, err
	}

	fields := models.ActivityFields{
		Action:     rbac.ActionDownload.String(),
		BucketID:   file.BucketID.String(),
		FileID:     file.ID.String(),
		ObjectType: rbac.ResourceFile.String(),
		UserID:     user.UserID.String(),
	}
	if versionNumber > 0 {
		fields.Version = strconv.Itoa(versionNumber)
	}

	action := models.Activity{
		Message: activity.FileDownloaded,
		Object:  file.ToActivity(),
		Filter:  activity.NewLogFilter(fields),
	}
	err = s.ActivityLogger.Send(action)
	if err != nil {
		return models.FileDownloadResponse{}, err
	}

	var bucket models.Bucket
	if dbErr := s.DB.Where("id = ?", file.BucketID).First(&bucket).Error; dbErr == nil {
		evt := events.NewFileActivityNotification(
			s.Publisher, events.FileActivityDownload, events.FileActivitySourceUser,
			file.BucketID, bucket.Name, file.Name, user.UserID, user.Email,
		)
		evt.Trigger()
	}

	return models.FileDownloadResponse{
		ID:  file.ID.String(),
		URL: url,
	}, nil
}

func (s BucketFileService) ListFileVersions(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) ([]models.FileVersionResponse, error) {
	file, err := sql.GetFileByID(s.DB, ids[0], ids[1])
	if err != nil {
		return nil, err
	}

	var versions []models.FileVersion
	if listErr := s.DB.Where("file_id = ? AND status = ?", file.ID, models.FileStatusUploaded).
		Order("version DESC").
		Find(&versions).Error; listErr != nil {
		logger.Error("Failed to list file versions", zap.Error(listErr))
		return nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	response := make([]models.FileVersionResponse, 0, len(versions))
	for _, version := range versions {
		response = append(response, models.FileVersionResponse{
			ID:         version.ID,
			Version:    version.Version,
			Size:       version.Size,
			Status:     version.Status,
			UploadedBy: version.UploadedBy,
			IsCurrent:  file.CurrentVersionID != nil && *file.CurrentVersionID == version.ID,
			CreatedAt:  version.CreatedAt,
		})
	}

	return response, nil
}

func (s BucketFileService) RestoreFileVersion(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileVersionRestoreBody,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", ids[1], ids[0]).
			First(&file).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			logger.Error("Failed to fetch file for version restore", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
			return apierrors.New(http.StatusForbidden, apierrors.CodeFileExpired)
		}

		version, err := sql.GetFileVersionByID(tx, file.ID, body.VersionID)
		if err != nil {
			return err
		}
		if version.Status != models.FileStatusUploaded {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotRestorable)
		}

		if updateErr := tx.Model(&models.File{}).
			Where("id = ?", file.ID).
			Updates(map[string]interface{}{
				"current_version_id": version.ID,
				"size":               version.Size,
			}).Error; updateErr != nil {
			logger.Error("Failed to repoint file to restored version", zap.Error(updateErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if logErr := s.ActivityLogger.Send(models.Activity{
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

func (s BucketFileService) DeleteFileVersion(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	var objectKey string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", ids[1], ids[0]).
			First(&file).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			logger.Error("Failed to fetch file for version deletion", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		version, err := sql.GetFileVersionByID(tx, file.ID, ids[2])
		if err != nil {
			return err
		}
		if version.Status != models.FileStatusUploaded {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotDeletable)
		}
		if file.CurrentVersionID != nil && *file.CurrentVersionID == version.ID {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionIsCurrent)
		}

		objectKey = sql.VersionObjectKey(file.BucketID, version.ID)

		if delErr := tx.Delete(&version).Error; delErr != nil {
			logger.Error("Failed to delete file version", zap.Error(delErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if logErr := s.ActivityLogger.Send(models.Activity{
			Message: activity.FileVersionDeleted,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
				Version:    strconv.Itoa(version.Version),
			}),
		}); logErr != nil {
			logger.Error("Failed to log version deletion activity", zap.Error(logErr))
			return logErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	if rmErr := s.Storage.RemoveObject(objectKey); rmErr != nil {
		logger.Error(
			"Failed to remove deleted version object - orphaned bytes",
			zap.Error(rmErr),
			zap.String("path", objectKey),
		)
	}

	return nil
}

func (s BucketFileService) TrashFile(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			logger.Error("Failed to fetch file for trashing", zap.Error(result.Error))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}

		if file.DeletedAt.Valid {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileAlreadyTrashed)
		}

		if file.Status != models.FileStatusUploaded {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
		}

		updates := map[string]interface{}{
			"status":     models.FileStatusDeleted,
			"deleted_by": user.UserID,
		}
		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to update file for trashing", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
		}

		if err := tx.Delete(&file).Error; err != nil {
			logger.Error("Failed to soft delete file", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		if err := s.Storage.MarkAsTrashed(objectPath, file); err != nil {
			logger.Error(
				"Failed to mark file as trashed - rolling back transaction",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()),
			)
			return err
		}

		action := models.Activity{
			Message: activity.FileTrashed,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionErase.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log trash activity", zap.Error(err))
			return err
		}

		return nil
	})
}

func (s BucketFileService) restoreParentFolders(
	tx *gorm.DB,
	logger *zap.Logger,
	folderID *uuid.UUID,
	bucketID uuid.UUID,
) ([]models.Folder, error) {
	return h.RestoreParentFolders(tx, logger, folderID, bucketID)
}

// unmarkRestoredFolders removes trash markers for restored folders.
// This must be called AFTER the transaction commits to avoid race conditions.
func (s BucketFileService) unmarkRestoredFolders(logger *zap.Logger, folders []models.Folder) {
	for _, folder := range folders {
		objectPath := path.Join("buckets", folder.BucketID.String(), folder.ID.String())
		if err := s.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			logger.Warn("Failed to unmark parent folder as trashed",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
			// Continue - folders exist only in DB
		}
	}
}

// RestoreFile recovers a file from trash with atomic status transition.
func (s BucketFileService) RestoreFile(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	var restoredFolders []models.Folder
	var restoredFile models.File

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ? AND deleted_at IS NOT NULL", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			logger.Error("Failed to fetch file for restoring", zap.Error(result.Error))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}

		if file.Status == models.FileStatusRestoring {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileRestoreInProgress)
		}

		folders, err := s.restoreParentFolders(tx, logger, file.FolderID, file.BucketID)
		if err != nil {
			return err
		}
		restoredFolders = folders

		var existingFile models.File
		query := tx.Where(
			"bucket_id = ? AND name = ? AND id != ?",
			file.BucketID, file.Name, file.ID,
		)
		if file.FolderID != nil {
			query = query.Where("folder_id = ?", file.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		conflictResult := query.Find(&existingFile)

		if conflictResult.RowsAffected > 0 {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileNameConflict)
		}

		updates := map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
			"status":     models.FileStatusUploaded,
		}

		if updateErr := tx.Unscoped().Model(&file).Updates(updates).Error; updateErr != nil {
			logger.Error("Failed to restore file", zap.Error(updateErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
		}

		restoredFile = file

		action := models.Activity{
			Message: activity.FileRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
		}
		if activityErr := s.ActivityLogger.Send(action); activityErr != nil {
			logger.Error("Failed to log restore activity", zap.Error(activityErr))
			return activityErr
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.unmarkRestoredFolders(logger, restoredFolders)

	objectPath := path.Join("buckets", restoredFile.BucketID.String(), restoredFile.ID.String())
	if storageErr := s.Storage.UnmarkAsTrashed(objectPath, restoredFile); storageErr != nil {
		logger.Warn(
			"Failed to unmark file as trashed (file already restored in DB)",
			zap.Error(storageErr),
			zap.String("path", objectPath),
			zap.String("file_id", restoredFile.ID.String()),
		)
		// Don't return error - the database is already updated
	}

	return nil
}

// PurgeFile permanently deletes a file from trash with atomic status transition.
func (s BucketFileService) PurgeFile(
	logger *zap.Logger,
	user models.UserClaims,
	bucketID, fileID uuid.UUID,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", fileID, bucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
			}
			logger.Error("Failed to fetch file for purging", zap.Error(result.Error))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}

		if !file.DeletedAt.Valid && file.Status != models.FileStatusUploading {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileNotInTrash)
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		if multipart, isMultipart, _ := cache.GetMultipartState(s.Cache, file.ID.String()); isMultipart {
			if err := s.Storage.AbortMultipartUpload(objectPath, multipart.UploadID); err != nil {
				logger.Warn("Failed to abort multipart upload",
					zap.Error(err),
					zap.String("path", objectPath))
			}
			if delErr := cache.DeleteMultipartState(s.Cache, file.ID.String()); delErr != nil {
				logger.Warn("Failed to delete multipart state from cache", zap.Error(delErr))
			}
		}

		if err := s.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			logger.Warn("Failed to delete trash marker",
				zap.Error(err),
				zap.String("path", objectPath))
		}

		if err := s.Storage.RemoveObject(objectPath); err != nil {
			logger.Warn("Failed to delete file from storage",
				zap.Error(err),
				zap.String("path", objectPath))
		}

		if err := tx.Unscoped().Delete(&file).Error; err != nil {
			logger.Error("Failed to hard delete file from database", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
		}

		action := models.Activity{
			Message: activity.FileDeleted,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log purge activity", zap.Error(err))
			return err
		}

		return nil
	})
}
