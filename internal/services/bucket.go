package services

import (
	"context"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	c "github.com/safebucket/safebucket/internal/configuration"
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
)

type BucketService struct {
	DB                 *gorm.DB
	Cache              cache.ICache
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
		With(m.ValidateQuery[models.ActivityQueryParams]).
		Get("/activity", handlers.GetOneWithQueryHandler(s.GetActivity))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/", handlers.GetOneHandler(s.GetBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			With(m.Validate[models.BucketCreateUpdateBody]).
			Patch("/", handlers.BodyHandler(s.UpdateBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			With(m.ValidateQuery[models.ActivityQueryParams]).
			Get("/activity", handlers.GetOneWithQueryHandler(s.GetBucketActivity))

		r.Mount("/members", BucketMemberService{
			DB:             s.DB,
			Providers:      s.Providers,
			Publisher:      s.Publisher,
			ActivityLogger: s.ActivityLogger,
			WebURL:         s.WebURL,
		}.Routes())

		r.Mount("/", BucketFileService{
			DB:                 s.DB,
			Cache:              s.Cache,
			Storage:            s.Storage,
			Publisher:          s.Publisher,
			ActivityLogger:     s.ActivityLogger,
			TrashRetentionDays: s.TrashRetentionDays,
		}.Routes())

		r.Mount("/trash", BucketTrashService{
			DB:        s.DB,
			Publisher: s.Publisher,
		}.Routes())

		r.Mount("/folders", BucketFolderService{
			DB:                 s.DB,
			Storage:            s.Storage,
			Publisher:          s.Publisher,
			ActivityLogger:     s.ActivityLogger,
			TrashRetentionDays: s.TrashRetentionDays,
		}.Routes())

		r.Mount("/shares", BucketShareService{
			DB:             s.DB,
			ActivityLogger: s.ActivityLogger,
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
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				ObjectType: rbac.ResourceBucket.String(),
				BucketID:   newBucket.ID.String(),
				UserID:     user.UserID.String(),
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
		return models.Bucket{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeCreateFailed)
	}

	return newBucket, nil
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
) (models.Bucket, error) {
	bucketID := ids[0]

	bucket, err := sql.GetBucketByID(s.DB, bucketID)
	if err != nil {
		return bucket, err
	}
	bucket.Files = []models.File{}
	bucket.Folders = []models.Folder{}

	now := time.Now()
	expirationTime := now.Add(-c.UploadPolicyExpirationInMinutes * time.Minute)

	var files []models.File
	if err = s.DB.Where(
		"bucket_id = ? AND (expires_at IS NULL OR expires_at > ?) AND (status = ? OR (status = ? AND created_at > ?))",
		bucketID,
		now,
		models.FileStatusUploaded,
		models.FileStatusUploading,
		expirationTime,
	).Find(&files).Error; err != nil {
		logger.Error("Failed to list files", zap.Error(err))
		return bucket, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	var folders []models.Folder
	if err = s.DB.Where("bucket_id = ? AND status = ?", bucketID, models.FolderStatusCreated).
		Find(&folders).Error; err != nil {
		logger.Error("Failed to list folders", zap.Error(err))
		return bucket, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	bucket.Files = files
	bucket.Folders = folders

	return bucket, nil
}

func (s BucketService) UpdateBucket(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.BucketCreateUpdateBody,
) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		bucket := models.Bucket{}
		result := tx.Where("id = ?", ids[0]).First(&bucket)
		if result.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
		}

		bucket.Name = body.Name
		if err := tx.Model(&bucket).Update("name", body.Name).Error; err != nil {
			return err
		}

		action := models.Activity{
			Message: activity.BucketUpdated,
			Object:  bucket.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionUpdate.String(),
				BucketID:   bucket.ID.String(),
				ObjectType: rbac.ResourceBucket.String(),
				UserID:     user.UserID.String(),
			}),
		}

		return s.ActivityLogger.Send(action)
	})
	if err != nil {
		logger.Error("Failed to update bucket", zap.Error(err))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
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
			return apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
		}

		if _, err := gorm.G[models.Bucket](tx).Where("id = ?", bucket.ID).Delete(context.Background()); err != nil {
			return err
		}

		if _, err := gorm.G[models.Invite](
			tx,
		).Where("bucket_id = ?", bucket.ID).
			Delete(context.Background()); err != nil {
			return err
		}

		action := models.Activity{
			Message: activity.BucketDeleted,
			Object:  bucket.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   bucket.ID.String(),
				ObjectType: rbac.ResourceBucket.String(),
				UserID:     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			return err
		}

		event := events.NewBucketPurge(s.Publisher, bucket.ID, user.UserID)
		event.Trigger()
		return nil
	})
	if err != nil {
		logger.Error("Failed to delete bucket", zap.Error(err))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeDeleteFailed)
	}

	return nil
}

func (s BucketService) GetActivity(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	query models.ActivityQueryParams,
) (models.Page[map[string]interface{}], error) {
	buckets := s.GetBucketList(logger, user, ids)

	var bucketIDs []string
	for _, bucket := range buckets {
		bucketIDs = append(bucketIDs, bucket.ID.String())
	}

	if len(bucketIDs) == 0 {
		return models.Page[map[string]interface{}]{Data: []map[string]interface{}{}}, nil
	}

	searchCriteria := map[string][]string{
		"object_type": {
			rbac.ResourceBucket.String(),
			rbac.ResourceFile.String(),
			rbac.ResourceFolder.String(),
			rbac.ResourceShare.String(),
		},
		"bucket_id": bucketIDs,
	}

	return h.SearchActivityPage(
		s.DB, logger, s.ActivityLogger,
		searchCriteria,
		query,
	)
}

func (s BucketService) GetBucketActivity(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	query models.ActivityQueryParams,
) (models.Page[map[string]any], error) {
	bucket, err := sql.GetBucketByID(s.DB, ids[0])
	if err != nil {
		return models.Page[map[string]any]{}, err
	}

	searchCriteria := map[string][]string{
		"object_type": {
			rbac.ResourceBucket.String(),
			rbac.ResourceFile.String(),
			rbac.ResourceFolder.String(),
			rbac.ResourceShare.String(),
		},
		"bucket_id": {bucket.ID.String()},
	}

	return h.SearchActivityPage(
		s.DB, logger, s.ActivityLogger,
		searchCriteria,
		query,
	)
}
