package helpers

import (
	"errors"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MoveBatch(
	db *gorm.DB,
	bucketID uuid.UUID,
	target models.OptionalID,
	itemIDs uuid.UUIDs,
	move func(tx *gorm.DB, targetFolderID *uuid.UUID, itemID uuid.UUID) error,
) (models.MoveResponse, error) {
	if !target.Set {
		return models.MoveResponse{}, apierrors.New(http.StatusBadRequest, apierrors.CodeFieldRequired)
	}

	results := make([]models.MoveResult, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		result := models.MoveResult{ID: itemID, Status: models.MoveStatusOK}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := lockTargetFolder(tx, bucketID, target.ID); err != nil {
				return err
			}
			return move(tx, target.ID, itemID)
		})
		if err != nil {
			result.Status = models.MoveStatusError
			result.Code = errorCode(err)
		}
		results = append(results, result)
	}

	return models.MoveResponse{Results: results}, nil
}

func lockTargetFolder(tx *gorm.DB, bucketID uuid.UUID, targetFolderID *uuid.UUID) error {
	if targetFolderID == nil {
		return nil
	}
	var targetFolder models.Folder
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND bucket_id = ?", *targetFolderID, bucketID).
		First(&targetFolder)
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeParentFolderNotFound)
	}
	return nil
}

func SameFolder(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func NameTakenInFolder(
	tx *gorm.DB,
	model any,
	bucketID uuid.UUID,
	name string,
	excludeID uuid.UUID,
	folderID *uuid.UUID,
) (bool, error) {
	query := tx.Model(model).Where("bucket_id = ? AND name = ? AND id != ?", bucketID, name, excludeID)
	if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func ValidateFolderMove(
	tx *gorm.DB,
	bucketID,
	folderID uuid.UUID,
	targetFolderID *uuid.UUID,
) error {
	current := targetFolderID
	for current != nil {
		if *current == folderID {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidMoveTarget)
		}

		var folder models.Folder
		if tx.Where("id = ? AND bucket_id = ?", *current, bucketID).First(&folder).RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeParentFolderNotFound)
		}
		current = folder.FolderID
	}

	return nil
}

func errorCode(err error) string {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return apierrors.CodeInternalServerError
}
