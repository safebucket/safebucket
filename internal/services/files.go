package services

import (
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	"api/internal/sql"
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"path"
	"path/filepath"
	"time"
)

type FileService struct {
	DB *gorm.DB
	S3 *minio.Client
}

func (s FileService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(h.Validate[models.FileTransferBody]).Post("/", handlers.CreateHandler(s.UploadFile))

	r.Route("/{id}", func(r chi.Router) {
		r.Delete("/", handlers.DeleteHandler(s.DeleteFile))
		r.Get("/download", handlers.GetOneHandler(s.DownloadFile))
	})
	return r
}

func (s FileService) UploadFile(body models.FileTransferBody) (models.FileTransferResponse, error) {
	bucketId, err := uuid.Parse(body.BucketId)
	if err != nil {
		return models.FileTransferResponse{}, errors.New("INVALID_BUCKET_ID")
	}

	bucket, err := sql.GetById[models.Bucket](s.DB, bucketId)
	if err != nil {
		return models.FileTransferResponse{}, errors.New("BUCKET_NOT_FOUND")
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

	url, err := s.S3.PresignedPutObject(
		context.Background(),
		"safebucket",
		path.Join("buckets", body.BucketId, file.Path, file.Name),
		time.Minute*15,
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

func (s FileService) DeleteFile(id uuid.UUID) error {
	file, err := sql.GetById[models.File](s.DB, id)
	if err != nil {
		return errors.New("FILE_NOT_FOUND")
	}

	err = s.S3.RemoveObject(
		context.Background(),
		"safebucket",
		path.Join("buckets", id.String(), file.Path, file.Name),
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		zap.L().Warn("File does not exist in storage", zap.Error(err))
	}

	result := s.DB.Delete(&file)
	if result.Error != nil {
		zap.L().Error("Failed to delete file", zap.Error(result.Error))
		return errors.New("FILE_DELETION_FAILED")
	}
	return nil
}

func (s FileService) DownloadFile(id uuid.UUID) (models.FileTransferResponse, error) {
	file, err := sql.GetById[models.File](s.DB, id)
	if err != nil {
		return models.FileTransferResponse{}, errors.New("FILE_NOT_FOUND")
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
