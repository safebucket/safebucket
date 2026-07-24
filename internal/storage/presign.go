package storage

import (
	"net/http"

	c "github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
)

func PresignUpload(
	logger *zap.Logger,
	store IStorage,
	objectPath string,
	size int,
	metadata map[string]string,
) (models.FileUploadResponse, error) {
	method := store.UploadMethod()

	switch method {
	case c.UploadMethodPost:
		url, body, err := store.PresignedPostPolicy(objectPath, size, metadata)
		if err != nil {
			logger.Error("Generate presigned POST policy failed", zap.Error(err))
			return models.FileUploadResponse{}, err
		}

		return models.FileUploadResponse{Method: c.UploadMethodPost, URL: url, Body: body}, nil
	default:
		logger.Error("Unsupported upload method", zap.String("method", method))

		return models.FileUploadResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}
}
