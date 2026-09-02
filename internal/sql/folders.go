package sql

import (
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFolderSubtree(db *gorm.DB, bucketID, rootID uuid.UUID) ([]models.Folder, error) {
	var folders []models.Folder
	err := db.Raw(`
		WITH RECURSIVE folder_subtree AS (
			SELECT *
			FROM folders
			WHERE bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at IS NULL
			UNION
			SELECT folders.*
			FROM folders
			JOIN folder_subtree ON folders.folder_id = folder_subtree.id
			WHERE folders.bucket_id = ? AND folders.status = ? AND folders.deleted_at IS NULL
		)
		SELECT * FROM folder_subtree`,
		bucketID, rootID, models.FolderStatusCreated,
		bucketID, models.FolderStatusCreated,
	).Scan(&folders).Error
	return folders, err
}
