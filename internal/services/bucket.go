package services

import (
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
	"context"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"path"
	"path/filepath"
	"time"
)

type BucketService struct {
	DB        *gorm.DB
	S3        *minio.Client
	Enforcer  *casbin.Enforcer
	Publisher *messaging.IPublisher
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionList, -1)).
		Get("/", handlers.GetListHandler(s.GetBucketList))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionCreate, -1)).
		With(m.Validate[models.BucketCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateBucket))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
			Get("/", handlers.GetOneHandler(s.GetBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionUpdate, 0)).
			With(m.Validate[models.Bucket]).
			Patch("/", handlers.UpdateHandler(s.UpdateBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDelete, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDelete, 0)).
			With(m.Validate[models.FileTransferBody]).Post("/files", handlers.CreateHandler(s.UploadFile))

		r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionUpload, 0)).
			With(m.Validate[models.FileTransferBody]).Post("/files", handlers.CreateHandler(s.UploadFile))

		r.Route("/files/{id1}", func(r chi.Router) {

			r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionUpload, 0)).
				With(m.Validate[models.UpdateFileBody]).
				Patch("/", handlers.UpdateHandler(s.UpdateFile))

			r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionErase, 0)).
				Delete("/", handlers.DeleteHandler(s.DeleteFile))

			r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDownload, 0)).
				Get("/download", handlers.GetOneHandler(s.DownloadFile))
		})
	})
	return r
}

func (s BucketService) CreateBucket(user models.UserClaims, _ uuid.UUIDs, body models.BucketCreateBody) (models.Bucket, error) {
	//TODO: Migrate to SQL transaction
	newBucket := models.Bucket{Name: body.Name}
	s.DB.Create(&newBucket)

	err := groups.InsertGroupBucketViewer(s.Enforcer, newBucket)
	if err != nil {
		return models.Bucket{}, err
	}
	err = groups.InsertGroupBucketContributor(s.Enforcer, newBucket)
	if err != nil {
		return models.Bucket{}, err
	}
	err = groups.InsertGroupBucketOwner(s.Enforcer, newBucket)
	if err != nil {
		return models.Bucket{}, err
	}

	err = groups.AddUserToOwners(s.Enforcer, newBucket, user.UserID.String())
	if err != nil {
		return models.Bucket{}, err
	}

	for _, shareWith := range body.ShareWith {
		var shareWithUser models.User

		result := s.DB.Where("email = ?", shareWith.Email).First(&shareWithUser)

		if result.RowsAffected > 0 {
			switch shareWith.Group {
			case "viewer":
				err = groups.AddUserToViewers(s.Enforcer, newBucket, shareWithUser.ID.String())
			case "contributor":
				err = groups.AddUserToContributors(s.Enforcer, newBucket, shareWithUser.ID.String())
			case "owner":
				err = groups.AddUserToOwners(s.Enforcer, newBucket, shareWithUser.ID.String())
			}

			if err != nil {
				return models.Bucket{}, err
			}

			event := events.NewBucketSharedWith(
				*s.Publisher,
				newBucket,
				user.Email,
				shareWith.Email,
			)
			event.Trigger()
		}
	}

	return newBucket, nil
}

func (s BucketService) GetBucketList(user models.UserClaims) []models.Bucket {
	var buckets []models.Bucket
	if !user.Valid() {
		zap.L().Warn(fmt.Sprintf("Invalid user claims %v", user.UserID.String()))
		return []models.Bucket{}
	}
	roles, err := s.Enforcer.GetImplicitRolesForUser(user.UserID.String(), c.DefaultDomain)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("Error retrieving roles %v", user.UserID.String()))
		return []models.Bucket{}
	}

	var bucketIDs []string

	for _, role := range roles {
		policies, _ := s.Enforcer.GetFilteredPolicy(0, c.DefaultDomain,
			role,
			rbac.ResourceBucket.String(),
			"",
			rbac.ActionRead.String())

		for _, policy := range policies {
			bucketIDs = append(bucketIDs, policy[3])
		}
	}
	_ = s.DB.Model(&models.Bucket{}).Where("id IN ?", bucketIDs).Find(&buckets) // Todo: cache result
	return buckets
}

func (s BucketService) GetBucket(_ models.UserClaims, ids uuid.UUIDs) (models.Bucket, error) {
	bucketId := ids[0]
	var bucket models.Bucket
	bucket.Files = []models.File{}

	result := s.DB.Where("id = ?", bucketId).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	} else {
		var files []models.File
		result = s.DB.Where("bucket_id = ?", bucketId).Find(&files)

		if result.RowsAffected > 0 {
			bucket.Files = files
		}
		return bucket, nil
	}
}

func (s BucketService) UpdateBucket(ids uuid.UUIDs, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: ids[0]}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	} else {
		return bucket, nil
	}
}

func (s BucketService) DeleteBucket(ids uuid.UUIDs) error {
	result := s.DB.Where("id = ?", ids[0]).Delete(&models.Bucket{})
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	} else {
		return nil
	}
}

func (s BucketService) UploadFile(_ models.UserClaims, ids uuid.UUIDs, body models.FileTransferBody) (models.FileTransferResponse, error) {
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

	err = sql.Create[*models.File](s.DB, file)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	policy := minio.NewPostPolicy()
	_ = policy.SetBucket("safebucket") //TODO: set var
	_ = policy.SetKey(path.Join("/buckets", bucket.ID.String(), file.Path, file.Name))
	_ = policy.SetContentLengthRange(int64(body.Size), int64(body.Size))
	_ = policy.SetExpires(time.Now().UTC().Add(15 * time.Minute))
	url, formData, err := s.S3.PresignedPostPolicy(context.Background(), policy)

	if err != nil {
		zap.L().Error("Generate presigned URL failed", zap.Error(err))
		return models.FileTransferResponse{}, err
	}

	return models.FileTransferResponse{
		ID:   file.ID.String(),
		Url:  url.String(),
		Body: formData,
	}, nil
}

func (s BucketService) UpdateFile(ids uuid.UUIDs, body models.UpdateFileBody) (models.File, error) {
	bucketId, fileId := ids[0], ids[1]

	file, err := sql.GetFileById(s.DB, bucketId, fileId)
	if err != nil {
		return file, err
	}

	if *body.Uploaded {
		_, err := s.S3.StatObject(
			context.Background(),
			"safebucket",
			path.Join("buckets", file.BucketId.String(), file.Path, file.Name),
			minio.StatObjectOptions{},
		)

		if err != nil {
			return models.File{}, errors.NewAPIError(400, "FILE_NOT_UPLOADED")
		}

		s.DB.Model(&file).Updates(body)
		return file, nil
	}

	return file, nil
}

func (s BucketService) DeleteFile(ids uuid.UUIDs) error {
	bucketId, fileId := ids[0], ids[1]

	file, err := sql.GetFileById(s.DB, bucketId, fileId)
	if err != nil {
		return err
	}

	err = s.S3.RemoveObject(
		context.Background(),
		"safebucket",
		path.Join("buckets", bucketId.String(), file.Path, file.Name),
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		zap.L().Warn("File does not exist in storage", zap.Error(err))
	}

	result := s.DB.Delete(&file)
	if result.Error != nil {
		zap.L().Error("Failed to delete file", zap.Error(result.Error))
		return errors.NewAPIError(500, "FILE_DELETION_FAILED")
	}
	return nil
}

func (s BucketService) DownloadFile(_ models.UserClaims, ids uuid.UUIDs) (models.FileTransferResponse, error) {
	bucketId, fileId := ids[0], ids[1]

	file, err := sql.GetFileById(s.DB, bucketId, fileId)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	url, err := s.S3.PresignedGetObject(
		context.Background(),
		"safebucket",
		path.Join("buckets", file.BucketId.String(), file.Path, file.Name),
		time.Minute*15,
		nil,
	)

	if err != nil {
		zap.L().Error("Generate presigned URL failed", zap.Error(err))
		return models.FileTransferResponse{}, err
	}

	return models.FileTransferResponse{
		ID:  file.ID.String(),
		Url: url.String(),
	}, nil
}
