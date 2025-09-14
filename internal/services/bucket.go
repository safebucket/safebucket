package services

import (
	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/groups"
	"api/internal/sql"
	"api/internal/storage"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketService struct {
	DB             *gorm.DB
	Storage        storage.IStorage
	Enforcer       *casbin.Enforcer
	Publisher      messaging.IPublisher
	Providers      c.Providers
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionList, -1)).
		Get("/", handlers.GetListHandler(s.GetBucketList))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionCreate, -1)).
		With(m.Validate[models.BucketCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateBucket))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionList, -1)).
		Get("/activity", handlers.GetListHandler(s.GetActivity))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
			Get("/", handlers.GetOneHandler(s.GetBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionUpdate, 0)).
			With(m.Validate[models.Bucket]).
			Patch("/", handlers.UpdateHandler(s.UpdateBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDelete, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionUpload, 0)).
			With(m.Validate[models.FileTransferBody]).
			Post("/files", handlers.CreateHandler(s.UploadFile))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
			Get("/activity", handlers.GetOneHandler(s.GetBucketActivity))

		r.Mount("/members", BucketMemberService{
			DB:             s.DB,
			Enforcer:       s.Enforcer,
			Providers:      s.Providers,
			Publisher:      s.Publisher,
			ActivityLogger: s.ActivityLogger,
			WebUrl:         s.WebUrl,
		}.Routes())

		r.Route("/files/{id1}", func(r chi.Router) {
			r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionErase, 0)).
				Delete("/", handlers.DeleteHandler(s.DeleteFile))

			r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDownload, 0)).
				Get("/download", handlers.GetOneHandler(s.DownloadFile))
		})
	})

	return r
}

func (s BucketService) CreateBucket(logger *zap.Logger, user models.UserClaims, _ uuid.UUIDs, body models.BucketCreateBody) (models.Bucket, error) {
	tx := s.DB.Begin()

	newBucket := models.Bucket{Name: body.Name}
	newBucket.CreatedBy = user.UserID
	res := tx.Create(&newBucket)

	if res.Error != nil {
		logger.Error("Failed to create bucket", zap.Error(res.Error))
		tx.Rollback()
		return models.Bucket{}, errors.ErrorCreateFailed
	}

	err := groups.InsertGroupBucketViewer(s.Enforcer, newBucket)
	if err != nil {
		logger.Error("Failed to create bucket group viewer", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, errors.ErrorCreateFailed
	}

	err = groups.InsertGroupBucketContributor(s.Enforcer, newBucket)
	if err != nil {
		logger.Error("Failed to create bucket group contributor", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, errors.ErrorCreateFailed
	}

	err = groups.InsertGroupBucketOwner(s.Enforcer, newBucket)
	if err != nil {
		logger.Error("Failed to create bucket group owner", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, errors.ErrorCreateFailed
	}

	err = groups.AddUserToOwners(s.Enforcer, newBucket, user.UserID.String())
	if err != nil {
		logger.Error("Failed to add user to group owner", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, errors.ErrorCreateFailed
	}

	action := models.Activity{
		Message: activity.BucketCreated,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionCreate.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceBucket.String(),
			"bucket_id":   newBucket.ID.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	err = s.ActivityLogger.Send(action)

	if err != nil {
		logger.Error("Failed to register activity", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, err
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		tx.Rollback()
		return models.Bucket{}, err
	}

	return newBucket, nil
}

func (s BucketService) GetBucketList(logger *zap.Logger, user models.UserClaims, _ uuid.UUIDs) []models.Bucket {
	var buckets []models.Bucket
	if !user.Valid() {
		logger.Warn(fmt.Sprintf("Invalid user claims %v", user.UserID.String()))
		return []models.Bucket{}
	}
	roles, err := s.Enforcer.GetImplicitRolesForUser(user.UserID.String(), c.DefaultDomain)

	if err != nil {
		logger.Warn(fmt.Sprintf("Error retrieving roles %v", user.UserID.String()))
		return []models.Bucket{}
	}

	var bucketIDs []string

	for _, role := range roles {
		policies, _ := s.Enforcer.GetFilteredPolicy(
			0, c.DefaultDomain, role, rbac.ResourceBucket.String(), "", rbac.ActionRead.String(),
		)

		for _, policy := range policies {
			bucketIDs = append(bucketIDs, policy[3])
		}
	}
	_ = s.DB.Model(&models.Bucket{}).Where("id IN ?", bucketIDs).Find(&buckets) // Todo: cache result
	return buckets
}

func (s BucketService) GetBucket(_ *zap.Logger, _ models.UserClaims, ids uuid.UUIDs) (models.Bucket, error) {
	bucketId := ids[0]
	var bucket models.Bucket
	bucket.Files = []models.File{}

	result := s.DB.Where("id = ?", bucketId).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	} else {
		var files []models.File
		// Filter out expired files that haven't been uploaded yet (only applies to files, not folders)
		expirationTime := time.Now().Add(-c.UploadPolicyExpirationInMinutes * time.Minute)
		result = s.DB.Where(
			"bucket_id = ? AND (type = 'folder' OR uploaded = true OR created_at > ?)", bucketId, expirationTime,
		).Find(&files)

		if result.RowsAffected > 0 {
			bucket.Files = files
		}
		return bucket, nil
	}
}

func (s BucketService) UpdateBucket(_ *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: ids[0]}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	} else {
		return bucket, nil
	}
}

func (s BucketService) DeleteBucket(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		bucket := models.Bucket{}
		result := tx.Where("id = ?", ids[0]).First(&bucket)

		if result.RowsAffected == 0 {
			return errors.NewAPIError(404, "BUCKET_NOT_FOUND")
		} else {
			// Soft delete bucket
			if _, err := gorm.G[models.Bucket](tx).Where("id = ?", bucket.ID).Delete(context.Background()); err != nil {
				return err
			}

			// Hard delete all invitations associated to the bucket
			if _, err := gorm.G[models.Invite](tx).Where("bucket_id = ?", bucket.ID).Delete(context.Background()); err != nil {
				return err
			}

			// Remove bucket groups
			if err := groups.RemoveGroupBucketViewer(s.Enforcer, bucket); err != nil {
				return err
			}

			if err := groups.RemoveGroupBucketContributor(s.Enforcer, bucket); err != nil {
				return err
			}

			if err := groups.RemoveGroupBucketOwner(s.Enforcer, bucket); err != nil {
				return err
			}

			// Remove associated grouping policies
			if err := groups.RemoveUsersFromViewers(s.Enforcer, bucket); err != nil {
				return err
			}

			if err := groups.RemoveUsersFromContributors(s.Enforcer, bucket); err != nil {
				return err
			}

			if err := groups.RemoveUsersFromOwners(s.Enforcer, bucket); err != nil {
				return err
			}

			action := models.Activity{
				Message: activity.BucketDeleted,
				Filter: activity.NewLogFilter(map[string]string{
					"action":      rbac.ActionDelete.String(),
					"bucket_id":   bucket.ID.String(),
					"domain":      c.DefaultDomain,
					"object_type": rbac.ResourceBucket.String(),
					"user_id":     user.UserID.String(),
				}),
			}

			if err := s.ActivityLogger.Send(action); err != nil {
				return err
			}

			// Trigger async file and folder deletion from root path
			event := events.NewObjectDeletion(s.Publisher, bucket, "/")
			event.Trigger()
			return nil
		}
	})

	if err != nil {
		logger.Error("Failed to delete bucket", zap.Error(err))
		return errors.ErrorDeleteFailed
	}

	return nil
}

func (s BucketService) UploadFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs, body models.FileTransferBody) (models.FileTransferResponse, error) {
	bucket, err := sql.GetById[models.Bucket](s.DB, ids[0])
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	extension := filepath.Ext(body.Name)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	file := &models.File{
		Name:      body.Name,
		Extension: extension,
		BucketId:  bucket.ID,
		Path:      body.Path,
		Type:      body.Type,
		Size:      body.Size,
	}

	tx := s.DB.Begin()

	err = sql.Create[*models.File](tx, file)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	url, formData, err := s.Storage.PresignedPostPolicy(
		path.Join("buckets", bucket.ID.String(), file.Path, file.Name),
		body.Size,
		map[string]string{
			"bucket_id": bucket.ID.String(),
			"file_id":   file.ID.String(),
			"user_id":   user.UserID.String(),
		},
	)

	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		tx.Rollback()
		return models.FileTransferResponse{}, err
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		return models.FileTransferResponse{}, errors.NewAPIError(500, "UPLOAD_FAILED")
	}

	return models.FileTransferResponse{
		ID:   file.ID.String(),
		Url:  url,
		Body: formData,
	}, nil
}

func (s BucketService) DeleteFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	bucketId, fileId := ids[0], ids[1]

	file, err := sql.GetFileById(s.DB, bucketId, fileId)
	if err != nil {
		return err
	}

	err = s.Storage.RemoveObject(path.Join("buckets", bucketId.String(), file.Path, file.Name))

	if err != nil {
		logger.Warn("File does not exist in storage", zap.Error(err))
	}

	result := s.DB.Delete(&file)
	if result.Error != nil {
		logger.Error("Failed to delete file", zap.Error(result.Error))
		return errors.NewAPIError(500, "FILE_DELETION_FAILED")
	}

	action := models.Activity{
		Message: activity.FileDeleted,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDelete.String(),
			"bucket_id":   bucketId.String(),
			"file_id":     fileId.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
			"user_id":     user.UserID.String(),
		}),
	}
	err = s.ActivityLogger.Send(action)

	if err != nil {
		return err
	}

	return nil
}

func (s BucketService) DownloadFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) (models.FileTransferResponse, error) {
	bucketId, fileId := ids[0], ids[1]

	file, err := sql.GetFileById(s.DB, bucketId, fileId)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	url, err := s.Storage.PresignedGetObject(path.Join("buckets", file.BucketId.String(), file.Path, file.Name))

	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		return models.FileTransferResponse{}, err
	}

	action := models.Activity{
		Message: activity.FileDownloaded,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDownload.String(),
			"bucket_id":   bucketId.String(),
			"file_id":     fileId.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFile.String(),
			"user_id":     user.UserID.String(),
		}),
	}
	err = s.ActivityLogger.Send(action)

	if err != nil {
		return models.FileTransferResponse{}, err
	}

	return models.FileTransferResponse{
		ID:  file.ID.String(),
		Url: url,
	}, nil
}

func (s BucketService) GetActivity(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) []map[string]interface{} {
	buckets := s.GetBucketList(logger, user, ids)

	var bucketIds []string
	for _, bucket := range buckets {
		bucketIds = append(bucketIds, bucket.ID.String())
	}

	if len(bucketIds) > 0 {
		searchCriteria := map[string][]string{
			"domain":      {c.DefaultDomain},
			"object_type": {rbac.ResourceBucket.String(), rbac.ResourceFile.String()},
			"bucket_id":   bucketIds,
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

func (s BucketService) GetBucketActivity(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) (models.Page[map[string]interface{}], error) {
	bucket, err := s.GetBucket(logger, user, ids)

	if err != nil {
		return models.Page[map[string]interface{}]{}, err
	}

	searchCriteria := map[string][]string{
		"domain":      {c.DefaultDomain},
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
