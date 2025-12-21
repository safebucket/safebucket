package services

import (
	"context"
	"strings"
	"time"

	"api/internal/activity"
	c "api/internal/configuration"
	apierrors "api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketService struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	Publisher          messaging.IPublisher
	Providers          c.Providers
	ActivityLogger     activity.IActivityLogger
	WebURL             string
	TrashRetentionDays int
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleGuest)).
		Get("/", handlers.GetListHandler(s.GetBucketList))

	r.With(m.AuthorizeRole(models.RoleUser)).
		With(m.Validate[models.BucketCreateUpdateBody]).
		Post("/", handlers.CreateHandler(s.CreateBucket))

	r.With(m.AuthorizeRole(models.RoleGuest)).
		Get("/activity", handlers.GetListHandler(s.GetActivity))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			With(m.ValidateQuery[models.BucketQueryParams]).
			Get("/", handlers.GetOneWithQueryHandler(s.GetBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			With(m.Validate[models.BucketCreateUpdateBody]).
			Patch("/", handlers.UpdateHandler(s.UpdateBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/activity", handlers.GetOneHandler(s.GetBucketActivity))

		r.Mount("/members", BucketMemberService{
			DB:             s.DB,
			Providers:      s.Providers,
			Publisher:      s.Publisher,
			ActivityLogger: s.ActivityLogger,
			WebURL:         s.WebURL,
		}.Routes())

		r.Mount("/", BucketFileService{
			DB:                 s.DB,
			Storage:            s.Storage,
			ActivityLogger:     s.ActivityLogger,
			TrashRetentionDays: s.TrashRetentionDays,
		}.Routes())

		r.Mount("/folders", BucketFolderService{
			DB:                 s.DB,
			Storage:            s.Storage,
			Publisher:          s.Publisher,
			ActivityLogger:     s.ActivityLogger,
			TrashRetentionDays: s.TrashRetentionDays,
		}.Routes())
	})

	return r
}

func (s BucketService) CreateBucket(
	logger *zap.Logger,
	user models.UserClaims,
	_ uuid.UUIDs,
	body models.BucketCreateUpdateBody,
) (models.Bucket, error) {
	var newBucket models.Bucket

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		newBucket = models.Bucket{Name: body.Name, CreatedBy: user.UserID}
		res := tx.Create(&newBucket)

		if res.Error != nil {
			logger.Error("Failed to create bucket", zap.Error(res.Error))
			return res.Error
		}

		err := rbac.CreateMembership(tx, user.UserID, newBucket.ID, models.GroupOwner)
		if err != nil {
			logger.Error("Failed to create owner membership", zap.Error(err))
			return err
		}

		action := models.Activity{
			Message: activity.BucketCreated,
			Object:  newBucket.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionCreate.String(),
				"object_type": rbac.ResourceBucket.String(),
				"bucket_id":   newBucket.ID.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		err = s.ActivityLogger.Send(action)
		if err != nil {
			logger.Error("Failed to register activity", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return models.Bucket{}, apierrors.ErrCreateFailed
	}

	return newBucket, nil
}

// buildFilePath constructs the full folder path for a file using Unscoped queries
// to handle trashed folders. Returns path in format "/Folder1/Folder2".
func (s BucketService) buildFilePath(folderID *uuid.UUID) string {
	if folderID == nil {
		return "/"
	}

	var pathSegments []string
	currentFolderID := folderID

	// Traverse up the folder hierarchy (max 100 levels to prevent infinite loops)
	for i := 0; i < 100 && currentFolderID != nil; i++ {
		var folder models.Folder
		// Use Unscoped to query trashed folders
		if err := s.DB.Unscoped().Where("id = ?", currentFolderID).First(&folder).Error; err != nil {
			break // Folder not found, stop traversal
		}

		// Prepend folder name to path
		pathSegments = append([]string{folder.Name}, pathSegments...)
		currentFolderID = folder.FolderID
	}

	if len(pathSegments) == 0 {
		return "/"
	}

	return "/" + strings.Join(pathSegments, "/")
}

func (s BucketService) GetBucketList(
	logger *zap.Logger,
	user models.UserClaims,
	_ uuid.UUIDs,
) []models.Bucket {
	var buckets []models.Bucket
	if !user.Valid() {
		logger.Warn("Invalid user claims", zap.String("user_id", user.UserID.String()))
		return []models.Bucket{}
	}

	memberships, err := rbac.GetUserBuckets(s.DB, user.UserID)
	if err != nil {
		logger.Error(
			"Error retrieving user buckets",
			zap.Error(err),
			zap.String("user_id", user.UserID.String()),
		)
		return []models.Bucket{}
	}

	var bucketIDs []uuid.UUID
	for _, membership := range memberships {
		bucketIDs = append(bucketIDs, membership.BucketID)
	}

	if len(bucketIDs) == 0 {
		return []models.Bucket{}
	}

	if err = s.DB.Where("id IN ?", bucketIDs).Find(&buckets).Error; err != nil {
		logger.Error("Error querying buckets", zap.Error(err))
		return []models.Bucket{}
	}

	return buckets
}

func (s BucketService) GetBucket(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	queryParams models.BucketQueryParams,
) (models.Bucket, error) {
	bucketID := ids[0]
	var bucket models.Bucket
	bucket.Files = []models.File{}
	bucket.Folders = []models.Folder{}

	result := s.DB.Where("id = ?", bucketID).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, apierrors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	// Determine the status filter based on query parameter
	// Default behavior (empty) shows active items (uploaded + uploading)
	status := queryParams.Status

	var files []models.File
	var folders []models.Folder

	switch status {
	case "deleted":
		fileResult := s.DB.Unscoped().
			Where(
				"bucket_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status != ?)",
				bucketID,
				models.FileStatusRestoring,
			).
			Order("deleted_at DESC").
			Find(&files)

		if fileResult.Error != nil {
			logger.Error("Failed to list trashed files", zap.Error(fileResult.Error))
			files = []models.File{}
		} else {
			// Compute original path for each trashed file
			for i := range files {
				files[i].OriginalPath = s.buildFilePath(files[i].FolderID)
			}
		}

		// Fetch trashed folders (use Unscoped to query soft-deleted items)
		// Trashed = deleted_at IS NOT NULL and status != restoring
		folderResult := s.DB.Unscoped().
			Where(
				"bucket_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status != ?)",
				bucketID,
				models.FileStatusRestoring,
			).
			Order("deleted_at DESC").
			Find(&folders)

		if folderResult.Error != nil {
			logger.Error("Failed to list trashed folders", zap.Error(folderResult.Error))
			folders = []models.Folder{}
		} else {
			// Compute original path for each trashed folder
			for i := range folders {
				folders[i].OriginalPath = s.buildFilePath(folders[i].FolderID)
			}
		}

	case "all":
		// Show all items regardless of status
		result = s.DB.Where("bucket_id = ?", bucketID).Find(&files)
		if result.RowsAffected > 0 {
			bucket.Files = files
		}

		result = s.DB.Where("bucket_id = ?", bucketID).Find(&folders)
		if result.RowsAffected > 0 {
			bucket.Folders = folders
		}

	case "uploading":
		// Show only items being uploaded
		expirationTime := time.Now().Add(-c.UploadPolicyExpirationInMinutes * time.Minute)
		result = s.DB.Where(
			"bucket_id = ? AND status = ? AND created_at > ?",
			bucketID,
			models.FileStatusUploading,
			expirationTime,
		).Find(&files)

		if result.RowsAffected > 0 {
			bucket.Files = files
		}

	case "uploaded":
		fallthrough
	default:
		// Show only active (non-soft-deleted) items
		// Filter out expired files that haven't been uploaded yet
		// GORM automatically excludes soft-deleted items (deleted_at IS NOT NULL)
		expirationTime := time.Now().Add(-c.UploadPolicyExpirationInMinutes * time.Minute)
		result = s.DB.Where(
			"bucket_id = ? AND (status = ? OR (status = ? AND created_at > ?))",
			bucketID,
			models.FileStatusUploaded,
			models.FileStatusUploading,
			expirationTime,
		).Find(&files)

		if result.RowsAffected > 0 {
			bucket.Files = files
		}

		// Get folders (GORM automatically excludes soft-deleted items)
		result = s.DB.Where("bucket_id = ?", bucketID).Find(&folders)

		if result.RowsAffected > 0 {
			bucket.Folders = folders
		}
	}

	bucket.Files = files
	bucket.Folders = folders

	return bucket, nil
}

func (s BucketService) UpdateBucket(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.BucketCreateUpdateBody,
) error {
	bucket := models.Bucket{ID: ids[0]}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return apierrors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}
	return nil
}

func (s BucketService) DeleteBucket(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		bucket := models.Bucket{}
		result := tx.Where("id = ?", ids[0]).First(&bucket)

		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "BUCKET_NOT_FOUND")
		}

		// Soft delete bucket (memberships will be cascade deleted by foreign key constraint)
		if _, err := gorm.G[models.Bucket](tx).Where("id = ?", bucket.ID).Delete(context.Background()); err != nil {
			return err
		}

		// Hard delete all invitations associated to the bucket
		if _, err := gorm.G[models.Invite](tx).Where("bucket_id = ?", bucket.ID).Delete(context.Background()); err != nil {
			return err
		}

		action := models.Activity{
			Message: activity.BucketDeleted,
			Object:  bucket.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionDelete.String(),
				"bucket_id":   bucket.ID.String(),
				"object_type": rbac.ResourceBucket.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			return err
		}

		// Trigger async bucket purge (files and folders)
		event := events.NewBucketPurge(s.Publisher, bucket.ID, user.UserID)
		event.Trigger()
		return nil
	})
	if err != nil {
		logger.Error("Failed to delete bucket", zap.Error(err))
		return apierrors.ErrDeleteFailed
	}

	return nil
}

func (s BucketService) GetActivity(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) []map[string]interface{} {
	buckets := s.GetBucketList(logger, user, ids)

	var bucketIDs []string
	for _, bucket := range buckets {
		bucketIDs = append(bucketIDs, bucket.ID.String())
	}

	if len(bucketIDs) > 0 {
		searchCriteria := map[string][]string{
			"object_type": {rbac.ResourceBucket.String(), rbac.ResourceFile.String()},
			"bucket_id":   bucketIDs,
		}

		history, err := s.ActivityLogger.Search(searchCriteria)
		if err != nil {
			logger.Error("Search history failed", zap.Error(err))
			return []map[string]interface{}{}
		}

		if len(history) == 0 {
			return []map[string]interface{}{}
		}

		return activity.EnrichActivity(s.DB, history)
	}

	return []map[string]interface{}{}
}

func (s BucketService) GetBucketActivity(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) (models.Page[map[string]interface{}], error) {
	bucket, err := s.GetBucket(logger, user, ids, models.BucketQueryParams{})
	if err != nil {
		return models.Page[map[string]interface{}]{}, err
	}

	searchCriteria := map[string][]string{
		"object_type": {rbac.ResourceBucket.String(), rbac.ResourceFile.String()},
		"bucket_id":   {bucket.ID.String()},
	}

	history, err := s.ActivityLogger.Search(searchCriteria)
	if err != nil {
		logger.Error("Search history failed", zap.Error(err))
		return models.Page[map[string]interface{}]{}, err
	}

	if len(history) == 0 {
		return models.Page[map[string]interface{}]{}, nil
	}

	enriched := activity.EnrichActivity(s.DB, history)

	return models.Page[map[string]interface{}]{Data: enriched}, nil
}
