package services

import (
	"net/http"
	"strings"
	"time"

	c "github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketTrashService struct {
	DB        *gorm.DB
	Publisher messaging.IPublisher
}

func (s BucketTrashService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
		Get("/", handlers.GetOneHandler(s.GetBucketTrash))

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.BucketTrashBody]).
		Post("/", handlers.BodyHandlerWithStatus(s.BulkTrash, http.StatusAccepted))

	return r
}

func (s BucketTrashService) BulkTrash(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.BucketTrashBody,
) error {
	bucketID := ids[0]
	body.FolderIDs = h.DedupeUUIDs(body.FolderIDs)
	body.FileIDs = h.DedupeUUIDs(body.FileIDs)

	if len(body.FolderIDs)+len(body.FileIDs) == 0 ||
		len(body.FolderIDs)+len(body.FileIDs) > c.TrashBatchLimit {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeInvalidValue)
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if txErr := s.validateTrashItems(tx, bucketID, body); txErr != nil {
			return txErr
		}

		now := time.Now()

		if txErr := sql.MarkFilesDeleting(tx, bucketID, body.FolderIDs, body.FileIDs, user.UserID, now); txErr != nil {
			logger.Error("Failed to mark files for trashing", zap.Error(txErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
		}

		if txErr := sql.MarkFoldersDeleting(tx, bucketID, body.FolderIDs, user.UserID, now); txErr != nil {
			logger.Error("Failed to mark folders for trashing", zap.Error(txErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
		}

		return nil
	})
	if err != nil {
		return err
	}

	event := events.NewItemsTrash(s.Publisher, bucketID, user.UserID)
	event.Trigger()

	logger.Info("Bulk trash accepted",
		zap.String("bucket_id", bucketID.String()),
		zap.Int("folders", len(body.FolderIDs)),
		zap.Int("files", len(body.FileIDs)),
		zap.String("user_id", user.UserID.String()))

	return nil
}

func (s BucketTrashService) validateTrashItems(
	db *gorm.DB,
	bucketID uuid.UUID,
	body models.BucketTrashBody,
) error {
	files := make([]models.File, 0, len(body.FileIDs))
	if len(body.FileIDs) > 0 {
		if err := db.Unscoped().Where("bucket_id = ? AND id IN ?", bucketID, body.FileIDs).
			Find(&files).Error; err != nil {
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}
		if len(files) != len(body.FileIDs) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
		}
		for _, file := range files {
			if file.DeletedAt.Valid {
				return apierrors.New(http.StatusConflict, apierrors.CodeItemAlreadyTrashed)
			}
			if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
				return apierrors.New(http.StatusForbidden, apierrors.CodeFileExpired)
			}
			if file.Status != models.FileStatusUploaded {
				return apierrors.New(http.StatusConflict, apierrors.CodeItemNotTrashable)
			}
		}
	}

	folders := make([]models.Folder, 0, len(body.FolderIDs))
	if len(body.FolderIDs) > 0 {
		if err := db.Unscoped().Where("bucket_id = ? AND id IN ?", bucketID, body.FolderIDs).
			Find(&folders).Error; err != nil {
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}
		if len(folders) != len(body.FolderIDs) {
			return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
		}
		for _, folder := range folders {
			if folder.Status == models.FolderStatusRestoring {
				return apierrors.New(http.StatusConflict, apierrors.CodeFolderRestoreInProgress)
			}
			if folder.DeletedAt.Valid {
				return apierrors.New(http.StatusConflict, apierrors.CodeItemAlreadyTrashed)
			}
			if folder.Status != models.FolderStatusCreated {
				return apierrors.New(http.StatusConflict, apierrors.CodeItemNotTrashable)
			}
		}
	}

	return nil
}

func (s BucketTrashService) GetBucketTrash(
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

	var allFolders []models.Folder
	if err = s.DB.Unscoped().Where("bucket_id = ?", bucketID).Find(&allFolders).Error; err != nil {
		logger.Error("Failed to list folders", zap.Error(err))
		return bucket, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	foldersByID := make(map[uuid.UUID]models.Folder, len(allFolders))
	trashedFolders := make([]models.Folder, 0)
	for _, folder := range allFolders {
		foldersByID[folder.ID] = folder
		if folder.DeletedAt.Valid &&
			(folder.Status == models.FolderStatusDeleted || folder.Status == models.FolderStatusRestoring) {
			trashedFolders = append(trashedFolders, folder)
		}
	}

	var files []models.File
	if err = s.DB.Unscoped().Where(
		"bucket_id = ? AND deleted_at IS NOT NULL AND status IN ? AND (expires_at IS NULL OR expires_at > ?)",
		bucketID,
		[]models.FileStatus{models.FileStatusDeleted, models.FileStatusRestoring},
		now,
	).Find(&files).Error; err != nil {
		logger.Error("Failed to list trashed files", zap.Error(err))
		return bucket, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	for i := range files {
		files[i].OriginalPath = buildFolderPath(files[i].FolderID, foldersByID)
	}

	for i := range trashedFolders {
		trashedFolders[i].OriginalPath = buildFolderPath(trashedFolders[i].FolderID, foldersByID)
	}

	bucket.Files = files
	bucket.Folders = trashedFolders

	return bucket, nil
}

func buildFolderPath(folderID *uuid.UUID, foldersByID map[uuid.UUID]models.Folder) string {
	if folderID == nil {
		return "/"
	}

	var pathSegments []string
	currentFolderID := folderID

	for i := 0; i < 100 && currentFolderID != nil; i++ {
		folder, exists := foldersByID[*currentFolderID]
		if !exists {
			break
		}
		pathSegments = append(pathSegments, folder.Name)
		currentFolderID = folder.FolderID
	}

	if len(pathSegments) == 0 {
		return "/"
	}
	for i, j := 0, len(pathSegments)-1; i < j; i, j = i+1, j-1 {
		pathSegments[i], pathSegments[j] = pathSegments[j], pathSegments[i]
	}

	return "/" + strings.Join(pathSegments, "/")
}
