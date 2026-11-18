package fileversions

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
)

func (m Manager) DeleteFiles(files ...models.File) error {
	toDelete, err := sql.FindPurgeableFiles(m.DB, files)
	if err != nil || len(toDelete) == 0 {
		return err
	}
	if err = m.removeObjects(toDelete); err != nil {
		return err
	}
	return sql.DeletePurgeableFiles(m.DB, toDelete)
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
