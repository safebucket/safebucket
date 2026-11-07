package fileversions

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
)

func (m Manager) DeleteFiles(files ...models.File) error {
	var toDelete []models.File
	if err := m.DB.Unscoped().Where("id IN ? AND status = ? AND deleted_at IS NOT NULL", models.FileIDs(files), models.FileStatusDeleting).
		Find(&toDelete).Error; err != nil {
		return err
	}

	if len(toDelete) == 0 {
		return nil
	}
	if err := m.removeObjects(toDelete); err != nil {
		return err
	}

	return m.DB.Unscoped().
		Where("id IN ? AND status = ? AND deleted_at IS NOT NULL", models.FileIDs(toDelete), models.FileStatusDeleting).
		Delete(&models.File{}).Error
}

func (m Manager) removeObjects(files []models.File) error {
	var versions []models.FileVersion
	if err := m.DB.Where("file_id IN ?", models.FileIDs(files)).Find(&versions).Error; err != nil {
		return err
	}

	versionsByFile := make(map[uuid.UUID][]models.FileVersion, len(files))
	for _, version := range versions {
		versionsByFile[version.FileID] = append(versionsByFile[version.FileID], version)
	}

	var keys []string
	for _, file := range files {
		fileVersions := versionsByFile[file.ID]
		if len(fileVersions) == 0 {
			fileVersions = []models.FileVersion{{ID: file.ID}}
		}
		for _, version := range fileVersions {
			if err := m.abortUpload(file, version); err != nil {
				return err
			}
			keys = append(keys, storage.VersionObjectKey(file.BucketID, version.ID))
		}
	}
	return m.Storage.RemoveObjects(keys)
}
