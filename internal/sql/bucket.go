package sql

import (
	"errors"
	"net/http"
	"slices"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetBucketByID(db *gorm.DB, bucketID uuid.UUID) (models.Bucket, error) {
	var bucket models.Bucket

	if err := db.Where("id = ?", bucketID).First(&bucket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Bucket{}, apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
		}
		return models.Bucket{}, err
	}

	return bucket, nil
}

func MarkFilesDeleting(
	tx *gorm.DB,
	bucketID uuid.UUID,
	folderIDs, fileIDs uuid.UUIDs,
	deletedBy uuid.UUID,
	at time.Time,
) error {
	folderScope := append(slices.Clone(folderIDs), uuid.Nil)
	fileScope := append(slices.Clone(fileIDs), uuid.Nil)

	return tx.Exec(`
		WITH RECURSIVE folder_tree (id) AS (
			SELECT id
			FROM folders
			WHERE bucket_id = ? AND id IN ? AND deleted_at IS NULL AND status = ?
			UNION
			SELECT child.id
			FROM folders AS child
			JOIN folder_tree AS parent ON child.folder_id = parent.id
			WHERE child.bucket_id = ? AND child.deleted_at IS NULL AND child.status = ?
		)
		UPDATE files
		SET status = ?, deleted_by = ?, updated_at = ?
		WHERE bucket_id = ?
		  AND deleted_at IS NULL
		  AND status IN ?
		  AND (id IN ? OR folder_id IN (SELECT id FROM folder_tree))
	`,
		bucketID, folderScope, models.FolderStatusCreated,
		bucketID, models.FolderStatusCreated,
		models.FileStatusDeleting, deletedBy, at,
		bucketID,
		[]models.FileStatus{models.FileStatusUploading, models.FileStatusUploaded},
		fileScope,
	).Error
}

func MarkFoldersDeleting(
	tx *gorm.DB,
	bucketID uuid.UUID,
	folderIDs uuid.UUIDs,
	deletedBy uuid.UUID,
	at time.Time,
) error {
	folderScope := append(slices.Clone(folderIDs), uuid.Nil)

	return tx.Exec(`
		WITH RECURSIVE folder_tree (id) AS (
			SELECT id
			FROM folders
			WHERE bucket_id = ? AND id IN ? AND deleted_at IS NULL AND status = ?
			UNION
			SELECT child.id
			FROM folders AS child
			JOIN folder_tree AS parent ON child.folder_id = parent.id
			WHERE child.bucket_id = ? AND child.deleted_at IS NULL AND child.status = ?
		)
		UPDATE folders
		SET status = ?, deleted_by = ?, updated_at = ?
		WHERE id IN (SELECT id FROM folder_tree)
	`,
		bucketID, folderScope, models.FolderStatusCreated,
		bucketID, models.FolderStatusCreated,
		models.FolderStatusDeleting, deletedBy, at,
	).Error
}
