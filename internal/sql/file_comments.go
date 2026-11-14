package sql

import (
	"net/http"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileComment(db *gorm.DB, fileID, commentID uuid.UUID) (models.FileComment, error) {
	var comment models.FileComment
	result := db.Where("id = ? AND file_id = ?", commentID, fileID).Find(&comment)
	if result.Error != nil {
		return models.FileComment{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.FileComment{}, apierrors.New(http.StatusNotFound, apierrors.CodeCommentNotFound)
	}

	return comment, nil
}

func CheckFileCommentable(file models.File) error {
	if file.Status == models.FileStatusUploading && !file.DeletedAt.Valid {
		return apierrors.New(http.StatusConflict, apierrors.CodeFileUploadInProgress)
	}
	expired := file.ExpiresAt != nil && !file.ExpiresAt.After(time.Now())
	if file.DeletedAt.Valid || file.Status != models.FileStatusUploaded || expired {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
	}
	return nil
}

func ListFileComments(db *gorm.DB, fileID uuid.UUID) ([]models.FileComment, error) {
	var comments []models.FileComment
	err := db.Where("file_id = ?", fileID).
		Preload("User", func(tx *gorm.DB) *gorm.DB {
			return tx.Select("id", "first_name", "last_name", "email")
		}).
		Order("created_at ASC, id ASC").
		Find(&comments).Error

	return comments, err
}

func ListActiveMemberIDs(db *gorm.DB, bucketID uuid.UUID) (map[uuid.UUID]bool, error) {
	var ids []uuid.UUID
	if err := db.Model(&models.Membership{}).
		Where("bucket_id = ?", bucketID).
		Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}

	members := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		members[id] = true
	}
	return members, nil
}
