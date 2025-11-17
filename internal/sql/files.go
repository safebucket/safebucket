package sql

import (
	"errors"

	customerrors "api/internal/errors"
	"api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileByID(db *gorm.DB, bucketID uuid.UUID, fileID uuid.UUID) (models.File, error) {
	var file models.File

	if err := db.Where("id = ? AND bucket_id = ?", fileID, bucketID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, customerrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
		return models.File{}, err
	}

	return file, nil
}

func FolderExistsAtPath(db *gorm.DB, bucketID uuid.UUID, folderPath string) (bool, error) {
	if folderPath == "/" || folderPath == "" {
		return true, nil
	}

	// Parse the path to get parent path and folder name
	// Example: "/documents/subfolder" -> parent="/documents", name="subfolder"
	// Example: "/documents" -> parent="/", name="documents"
	lastSlashIndex := -1
	for i := len(folderPath) - 1; i >= 0; i-- {
		if folderPath[i] == '/' {
			lastSlashIndex = i
			break
		}
	}

	if lastSlashIndex == -1 {
		// No slash found, invalid path
		return false, nil
	}

	var parentPath string
	var folderName string

	if lastSlashIndex == 0 {
		// Path is like "/documents"
		parentPath = "/"
		folderName = folderPath[1:]
	} else {
		// Path is like "/documents/subfolder"
		parentPath = folderPath[:lastSlashIndex]
		folderName = folderPath[lastSlashIndex+1:]
	}

	var folder models.File
	result := db.Where(
		"bucket_id = ? AND path = ? AND name = ? AND type = ?",
		bucketID,
		parentPath,
		folderName,
		models.FileTypeFolder,
	).Find(&folder)

	if result.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}
