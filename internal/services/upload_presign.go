package services

import (
	"net/http"

	c "github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"go.uber.org/zap"
)

func presignUpload(
	logger *zap.Logger,
	store storage.IStorage,
	objectPath string,
	size int64,
	metadata map[string]string,
) (models.FileTransferResponse, error) {
	method := store.UploadMethod()

	switch method {
	case c.UploadMethodPost:
		url, body, err := store.PresignedPostPolicy(objectPath, int(size), metadata)
		if err != nil {
			logger.Error("Generate presigned POST policy failed", zap.Error(err))
			return models.FileTransferResponse{}, err
		}

		return models.FileTransferResponse{Method: c.UploadMethodPost, URL: url, Body: body}, nil
	default:
		logger.Error("Unsupported upload method", zap.String("method", method))

		return models.FileTransferResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}
}
