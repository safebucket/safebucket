package services

import (
	"errors"
	"net/http"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketMoveService struct {
	DB *gorm.DB
}

func (s BucketMoveService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.MoveBody]).
		Post("/", handlers.BatchHandler(s.MoveItems))

	return r
}

func (s BucketMoveService) MoveItems(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.MoveBody,
) (models.MoveResponse, error) {
	if !body.DestinationFolderID.Set {
		return models.MoveResponse{}, apierrors.New(http.StatusBadRequest, apierrors.CodeFieldRequired)
	}

	response := models.MoveResponse{}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		bucketID := ids[0]
		files, folders, err := lockItemsForMove(logger, tx, bucketID, body)
		if err != nil {
			return err
		}
		if err = validateMove(files, folders); err != nil {
			return err
		}

		targetID := body.DestinationFolderID.ID
		filePlan, duplicate := planMove(files, targetID, func(file models.File) moveItem {
			return moveItem{ID: file.ID, Name: file.Name, FolderID: file.FolderID}
		})
		if duplicate {
			return apierrors.New(http.StatusConflict, apierrors.CodeNameConflict)
		}
		folderPlan, duplicate := planMove(folders, targetID, func(folder models.Folder) moveItem {
			return moveItem{ID: folder.ID, Name: folder.Name, FolderID: folder.FolderID}
		})
		if duplicate {
			return apierrors.New(http.StatusConflict, apierrors.CodeNameConflict)
		}

		invalidTarget, err := sql.IsInvalidMoveTarget(tx, bucketID, targetID, folderPlan.IDs)
		if err != nil {
			return internalServerError(logger, "Failed to validate move destination", err)
		}
		if invalidTarget {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidMoveTarget)
		}

		hasFileConflict, err := sql.HasSameFileNamesAtTarget(tx, bucketID, targetID, filePlan.Names)
		if err != nil {
			return internalServerError(logger, "Failed to check file move conflicts", err)
		}
		if hasFileConflict {
			return apierrors.New(http.StatusConflict, apierrors.CodeNameConflict)
		}

		hasFolderConflict, err := sql.HasSameFolderNamesAtTarget(tx, bucketID, targetID, folderPlan.Names)
		if err != nil {
			return internalServerError(logger, "Failed to check folder move conflicts", err)
		}
		if hasFolderConflict {
			return apierrors.New(http.StatusConflict, apierrors.CodeNameConflict)
		}

		movedFiles, err := sql.MoveFiles(tx, bucketID, filePlan.IDs, targetID)
		if err != nil {
			return internalServerError(logger, "Failed to move files", err)
		}
		movedFolders, err := sql.MoveFolders(tx, bucketID, folderPlan.IDs, targetID)
		if err != nil {
			return internalServerError(logger, "Failed to move folders", err)
		}

		response = models.MoveResponse{
			MovedFiles:     int(movedFiles),
			MovedFolders:   int(movedFolders),
			UnchangedItems: filePlan.Unchanged + folderPlan.Unchanged,
		}
		return nil
	})
	if err != nil {
		return models.MoveResponse{}, err
	}
	return response, nil
}

func lockItemsForMove(
	logger *zap.Logger,
	tx *gorm.DB,
	bucketID uuid.UUID,
	body models.MoveBody,
) ([]models.File, []models.Folder, error) {
	if err := sql.LockBucketForUpdate(tx, bucketID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apierrors.New(http.StatusNotFound, apierrors.CodeItemNotFound)
		}
		return nil, nil, internalServerError(logger, "Failed to lock bucket for moving", err)
	}

	target, err := sql.LockMoveTarget(tx, bucketID, body.DestinationFolderID.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apierrors.New(http.StatusNotFound, apierrors.CodeItemNotFound)
		}
		return nil, nil, internalServerError(logger, "Failed to lock move destination", err)
	}
	if target != nil && target.Status != models.FolderStatusCreated {
		return nil, nil, apierrors.New(http.StatusConflict, apierrors.CodeItemNotMovable)
	}

	files, err := sql.LockFilesForMove(tx, bucketID, body.FileIDs)
	if err != nil {
		return nil, nil, internalServerError(logger, "Failed to lock files for moving", err)
	}
	if len(files) != len(body.FileIDs) {
		return nil, nil, apierrors.New(http.StatusNotFound, apierrors.CodeItemNotFound)
	}

	folders, err := sql.LockFoldersForMove(tx, bucketID, body.FolderIDs)
	if err != nil {
		return nil, nil, internalServerError(logger, "Failed to lock folders for moving", err)
	}
	if len(folders) != len(body.FolderIDs) {
		return nil, nil, apierrors.New(http.StatusNotFound, apierrors.CodeItemNotFound)
	}

	return files, folders, nil
}

func validateMove(files []models.File, folders []models.Folder) error {
	now := time.Now()
	for _, file := range files {
		if file.Status != models.FileStatusUploading && file.Status != models.FileStatusUploaded {
			return apierrors.New(http.StatusConflict, apierrors.CodeItemNotMovable)
		}
		if file.ExpiresAt != nil && file.ExpiresAt.Before(now) {
			return apierrors.New(http.StatusConflict, apierrors.CodeItemNotMovable)
		}
	}

	for _, folder := range folders {
		if folder.Status != models.FolderStatusCreated {
			return apierrors.New(http.StatusConflict, apierrors.CodeItemNotMovable)
		}
	}

	return nil
}

func internalServerError(logger *zap.Logger, message string, err error) error {
	logger.Error(message, zap.Error(err))
	return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
}

func sameFolder(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type movePlan struct {
	IDs       uuid.UUIDs
	Names     []string
	Unchanged int
}

type moveItem struct {
	ID       uuid.UUID
	Name     string
	FolderID *uuid.UUID
}

func planMove[T any](items []T, targetID *uuid.UUID, toMoveItem func(T) moveItem) (movePlan, bool) {
	plan := movePlan{IDs: make(uuid.UUIDs, 0, len(items)), Names: make([]string, 0, len(items))}
	seen := make(map[string]struct{}, len(items))
	for _, value := range items {
		item := toMoveItem(value)
		if sameFolder(item.FolderID, targetID) {
			plan.Unchanged++
			continue
		}
		if _, exists := seen[item.Name]; exists {
			return movePlan{}, true
		}
		seen[item.Name] = struct{}{}
		plan.IDs = append(plan.IDs, item.ID)
		plan.Names = append(plan.Names, item.Name)
	}
	return plan, false
}
