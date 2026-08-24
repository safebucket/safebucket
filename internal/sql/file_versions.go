package sql

import (
	"path"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func VersionObjectKey(bucketID, versionID uuid.UUID) string {
	return path.Join("buckets", bucketID.String(), versionID.String())
}

// NextVersionNumber requires the caller to hold the file row's write lock, or to have created it in
// the same transaction. The number derives from MAX(version_number), so it repeats after the last
// version is deleted; activity timestamps disambiguate reused numbers.
func NextVersionNumber(tx *gorm.DB, file *models.File) (int, error) {
	var current int
	err := tx.Model(&models.FileVersion{}).
		Where("file_id = ?", file.ID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&current).Error
	if err != nil {
		return 0, err
	}

	return current + 1, nil
}

func PendingVersion(tx *gorm.DB, fileID uuid.UUID) (models.FileVersion, error) {
	var version models.FileVersion
	err := tx.Where("file_id = ? AND status = ?", fileID, models.FileStatusUploading).
		Order("version_number DESC").
		First(&version).Error

	return version, err
}

func ListVersionObjectKeys(db *gorm.DB, bucketID, fileID uuid.UUID) ([]string, error) {
	var versions []models.FileVersion
	if err := db.Where("file_id = ?", fileID).Find(&versions).Error; err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(versions))
	for _, v := range versions {
		keys = append(keys, VersionObjectKey(bucketID, v.ID))
	}

	return keys, nil
}

func ListVersionObjectKeysForFiles(db *gorm.DB, fileIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	keysByFile := make(map[uuid.UUID][]string, len(fileIDs))
	if len(fileIDs) == 0 {
		return keysByFile, nil
	}

	type versionRow struct {
		ID       uuid.UUID
		FileID   uuid.UUID
		BucketID uuid.UUID
	}

	var rows []versionRow
	if err := db.
		Model(&models.FileVersion{}).
		Select("file_versions.id AS id, file_versions.file_id AS file_id, files.bucket_id AS bucket_id").
		Joins("JOIN files ON files.id = file_versions.file_id").
		Where("file_versions.file_id IN ?", fileIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		keysByFile[row.FileID] = append(keysByFile[row.FileID], VersionObjectKey(row.BucketID, row.ID))
	}

	return keysByFile, nil
}

// PromoteVersion returns the object keys of the pruned versions, for the caller to remove after the
// transaction commits, and whether this call performed the promotion (false if another confirm path
// got there first).
func PromoteVersion(
	tx *gorm.DB,
	file *models.File,
	versionID uuid.UUID,
	maxVersions int,
) ([]string, bool, error) {
	var version models.FileVersion
	if err := tx.Where("id = ? AND file_id = ?", versionID, file.ID).First(&version).Error; err != nil {
		return nil, false, err
	}

	result := tx.Model(&models.FileVersion{}).
		Where("id = ? AND status = ?", versionID, models.FileStatusUploading).
		Update("status", models.FileStatusUploaded)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		// Already promoted by the other confirm path: no-op, so a duplicate confirmation neither
		// re-syncs the pointer nor re-runs pruning.
		return nil, false, nil
	}

	if err := tx.Model(&models.File{}).
		Where("id = ?", file.ID).
		Updates(map[string]interface{}{
			"current_version_id": versionID,
			"size":               version.Size,
			"status":             models.FileStatusUploaded,
		}).Error; err != nil {
		return nil, false, err
	}

	file.CurrentVersionID = &versionID
	file.Size = version.Size
	file.Status = models.FileStatusUploaded

	prunedKeys, err := pruneOldVersions(tx, file.BucketID, file.ID, versionID, maxVersions)
	return prunedKeys, true, err
}

func pruneOldVersions(
	tx *gorm.DB,
	bucketID, fileID, currentVersionID uuid.UUID,
	maxVersions int,
) ([]string, error) {
	var versions []models.FileVersion
	if err := tx.Where("file_id = ? AND status = ?", fileID, models.FileStatusUploaded).
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		return nil, err
	}

	if len(versions) <= maxVersions {
		return nil, nil
	}

	prunedKeys := make([]string, 0, len(versions)-maxVersions)
	prunedIDs := make([]uuid.UUID, 0, len(versions)-maxVersions)
	for i, v := range versions {
		if i < maxVersions || v.ID == currentVersionID {
			continue
		}
		prunedIDs = append(prunedIDs, v.ID)
		prunedKeys = append(prunedKeys, VersionObjectKey(bucketID, v.ID))
	}

	if len(prunedIDs) == 0 {
		return nil, nil
	}

	if err := tx.Where("id IN ?", prunedIDs).Delete(&models.FileVersion{}).Error; err != nil {
		return nil, err
	}

	return prunedKeys, nil
}
