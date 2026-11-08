package sql

import (
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFolderSubtree(db *gorm.DB, bucketID, rootID uuid.UUID) ([]models.Folder, error) {
	folders := make([]models.Folder, 0)
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

func FindTrashedFilesInTree(db *gorm.DB, bucketID, rootID uuid.UUID, limit int) ([]models.File, error) {
	files := make([]models.File, 0)
	err := db.Raw(`
    	WITH RECURSIVE folder_tree (id) AS (
            SELECT id
            FROM folders
            WHERE bucket_id = ? AND id = ?
            UNION
            SELECT child.id
            FROM folders AS child
            JOIN folder_tree AS parent ON child.folder_id = parent.id
            WHERE child.bucket_id = ? AND child.deleted_at IS NOT NULL
		)
        SELECT *
        FROM files
        WHERE bucket_id = ? AND deleted_at IS NOT NULL AND folder_id IN (SELECT id FROM folder_tree)
        LIMIT ?`,
		bucketID, rootID, bucketID, bucketID, limit,
	).Scan(&files).Error
	return files, err
}

func DeleteTrashedFoldersInTree(tx *gorm.DB, bucketID, rootID uuid.UUID, limit int) (int64, error) {
	result := tx.Exec(`
		WITH RECURSIVE folder_tree (id) AS (
			SELECT id
			FROM folders
			WHERE bucket_id = ? AND id = ?
			UNION
			SELECT child.id
			FROM folders AS child
			JOIN folder_tree AS parent ON child.folder_id = parent.id
			WHERE child.bucket_id = ? AND child.deleted_at IS NOT NULL
		)
		DELETE FROM folders
		WHERE id IN (
			SELECT id
			FROM folder_tree
			WHERE id != ?
			  AND NOT EXISTS (SELECT 1 FROM folders AS child WHERE child.folder_id = folder_tree.id)
			LIMIT ?
		)`,
		bucketID, rootID, bucketID, rootID, limit,
	)
	return result.RowsAffected, result.Error
}
