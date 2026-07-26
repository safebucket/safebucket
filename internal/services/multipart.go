package services

import (
	"net/http"

	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/storage"

	"go.uber.org/zap"
)

func storeMultipartState(
	logger *zap.Logger,
	cacheClient cache.ICache,
	store storage.IStorage,
	fileID, objectPath, uploadID string,
	partSize int64,
) error {
	if uploadID == "" {
		return nil
	}

	state := cache.MultipartState{UploadID: uploadID, PartSize: partSize}
	if err := cache.SetMultipartState(cacheClient, fileID, state); err != nil {
		if abortErr := store.AbortMultipartUpload(objectPath, uploadID); abortErr != nil {
			logger.Warn("Failed to abort orphaned multipart upload", zap.Error(abortErr))
		}
		return err
	}

	return nil
}

func completeMultipartUpload(
	logger *zap.Logger,
	store storage.IStorage,
	objectPath, uploadID string,
	partSize int64,
	fileSize int,
) error {
	parts, err := store.ListUploadedParts(objectPath, uploadID)
	if err != nil {
		logger.Error("Failed to list uploaded parts", zap.Error(err), zap.String("path", objectPath))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
	}

	partCount := storage.PartCount(int64(fileSize), partSize)
	if len(parts) != partCount {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	}

	var total int64
	for _, part := range parts {
		expected := storage.ExpectedPartSize(int64(fileSize), partSize, part.PartNumber, partCount)
		if part.Size != expected {
			return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
		}
		total += part.Size
	}
	if total != int64(fileSize) {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	}

	if completeErr := store.CompleteMultipartUpload(objectPath, uploadID, parts); completeErr != nil {
		logger.Error("Failed to complete multipart upload", zap.Error(completeErr), zap.String("path", objectPath))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
	}

	return nil
}
