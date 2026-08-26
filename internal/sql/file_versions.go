package sql

import (
	"errors"
	"net/http"
	"path"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func VersionObjectKey(bucketID, versionID uuid.UUID) string {
	return path.Join("buckets", bucketID.String(), versionID.String())
}

func GetFileVersionByID(db *gorm.DB, fileID, versionID uuid.UUID) (models.FileVersion, error) {
	var version models.FileVersion
	if err := db.Where("id = ? AND file_id = ?", versionID, fileID).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.FileVersion{}, apierrors.New(http.StatusNotFound, apierrors.CodeFileVersionNotFound)
		}
		return models.FileVersion{}, err
	}

	return version, nil
}
