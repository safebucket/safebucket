package helpers

import (
	"errors"
	"maps"
	"net/http"

	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ConfirmVersionObject(
	c cache.ICache,
	store storage.IStorage,
	logger *zap.Logger,
	objectPath string,
	version models.FileVersion,
	metadata map[string]string,
) error {
	multipart, isMultipart, cacheErr := cache.GetMultipartState(c, version.ID.String())
	if cacheErr != nil {
		logger.Error("Failed to read multipart state", zap.Error(cacheErr))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	if isMultipart {
		if completeErr := storage.FinalizeMultipartUpload(
			store, objectPath, multipart.UploadID, multipart.PartSize, int64(version.Size), metadata,
		); completeErr != nil {
			if errors.Is(completeErr, storage.ErrMultipartPartMismatch) {
				return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
			}
			logger.Error("Failed to complete multipart upload",
				zap.Error(completeErr), zap.String("path", objectPath))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
		}
	}

	if _, statErr := store.StatObject(objectPath); statErr != nil {
		logger.Error("File not found in storage", zap.Error(statErr), zap.String("path", objectPath))
		return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotInStorage)
	}

	return nil
}

// ConfirmPendingVersion requires the caller to already hold the file row's write lock. The returned
// object keys, from the versions retention pruned, must be removed once the transaction commits,
// not before.
func ConfirmPendingVersion(
	tx *gorm.DB,
	c cache.ICache,
	store storage.IStorage,
	logger *zap.Logger,
	file *models.File,
	extraMetadata map[string]string,
	maxVersions int,
) (models.FileVersion, []string, error) {
	version, err := sql.PendingVersion(tx, file.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Nothing awaiting confirmation: a duplicate confirm, or the storage-notification path
			// finalized it first.
			return version, nil, apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
		}
		logger.Error("Failed to read pending version", zap.Error(err))
		return version, nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	// Derived identity is written last so a caller cannot override it: the event parser reads this
	// metadata back off the object to route upload confirmations, so a wrong file_id here would
	// promote a version of the wrong file.
	metadata := make(map[string]string, len(extraMetadata)+3)
	maps.Copy(metadata, extraMetadata)
	metadata["bucket_id"] = file.BucketID.String()
	metadata["file_id"] = file.ID.String()
	metadata["version_id"] = version.ID.String()

	objectPath := sql.VersionObjectKey(file.BucketID, version.ID)
	if confirmErr := ConfirmVersionObject(c, store, logger, objectPath, version, metadata); confirmErr != nil {
		return version, nil, confirmErr
	}

	prunedKeys, promoted, promoteErr := sql.PromoteVersion(tx, file, version.ID, maxVersions)
	if promoteErr != nil {
		logger.Error("Failed to promote file version", zap.Error(promoteErr))
		return version, nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	if !promoted {
		// Unreachable: every promote path takes the file row's write lock first.
		logger.Error("version promotion unexpectedly no-op'd", zap.String("version_id", version.ID.String()))
		return version, nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	if delErr := cache.DeleteMultipartState(c, version.ID.String()); delErr != nil {
		logger.Warn("Failed to delete multipart state from cache", zap.Error(delErr))
	}

	return version, prunedKeys, nil
}

// PurgeFilesWithVersions deletes the version rows explicitly rather than leaving them to the
// foreign key cascade, which SQLite does not reliably enforce.
func PurgeFilesWithVersions(
	tx *gorm.DB,
	store storage.IStorage,
	logger *zap.Logger,
	fileIDs []uuid.UUID,
) error {
	if len(fileIDs) == 0 {
		return nil
	}

	keysByFile, err := sql.ListVersionObjectKeysForFiles(tx, fileIDs)
	if err != nil {
		logger.Error("Failed to list version keys for purging", zap.Error(err))
		return err
	}

	var objectKeys []string
	for _, keys := range keysByFile {
		objectKeys = append(objectKeys, keys...)
	}

	RemoveOrphanedVersionObjects(store, logger, objectKeys)

	if err = tx.Where("file_id IN ?", fileIDs).Delete(&models.FileVersion{}).Error; err != nil {
		logger.Error("Failed to delete file versions from database", zap.Error(err))
		return err
	}

	if err = tx.Unscoped().Where("id IN ?", fileIDs).Delete(&models.File{}).Error; err != nil {
		logger.Error("Failed to delete files from database", zap.Error(err))
		return err
	}

	return nil
}

// RemoveOrphanedVersionObjects is for paths with no retry available, because the rows naming the
// keys are already gone. Anything left is logged at error level: that line is the only remaining
// record of bytes still sitting in the bucket.
func RemoveOrphanedVersionObjects(store storage.IStorage, logger *zap.Logger, keys []string) {
	if failed := RemoveVersionObjects(store, logger, keys); len(failed) > 0 {
		logger.Error("Objects left in storage with no database record",
			zap.Strings("paths", failed), zap.Int("total", len(keys)))
	}
}

// RemoveVersionObjects removes each key independently and returns those still present afterwards.
// IStorage.RemoveObjects aborts on the first failing key, which would strand every other version's
// object behind one unfinished version whose bytes never landed. A key already absent counts as
// removed, since providers disagree on whether deleting a missing object is an error.
func RemoveVersionObjects(store storage.IStorage, logger *zap.Logger, keys []string) []string {
	var failed []string

	for _, key := range keys {
		err := store.RemoveObject(key)
		if err == nil {
			continue
		}

		if _, statErr := store.StatObject(key); statErr != nil {
			continue
		}

		logger.Warn("Failed to remove version object", zap.Error(err), zap.String("path", key))
		failed = append(failed, key)
	}

	return failed
}
