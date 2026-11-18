package fileversions

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	DB          *gorm.DB
	Storage     storage.IStorage
	MaxVersions int
}

type Completion struct {
	File    models.File
	Version models.FileVersion
	Changed bool
}

func Metadata(file models.File, version models.FileVersion, share *models.Share) map[string]string {
	metadata := map[string]string{
		"bucket_id":  file.BucketID.String(),
		"file_id":    file.ID.String(),
		"version_id": version.ID.String(),
	}
	if version.UploadedBy != nil {
		metadata["user_id"] = version.UploadedBy.String()
	}
	if share != nil {
		metadata["share_id"] = share.ID.String()
	}
	return metadata
}

func (m Manager) abortUpload(file models.File, version models.FileVersion) error {
	if version.UploadID == "" {
		return nil
	}
	return m.Storage.AbortMultipartUpload(storage.VersionObjectKey(file.BucketID, version.ID), version.UploadID)
}

func isCurrent(file models.File, versionID uuid.UUID) bool {
	return file.CurrentVersionID != nil && *file.CurrentVersionID == versionID
}

func checkShareUpload(db *gorm.DB, share *models.Share, file models.File) error {
	if share == nil {
		return nil
	}
	if !share.AllowUpload {
		return apierrors.New(http.StatusForbidden, apierrors.CodeShareUploadNotAllowed)
	}
	if !helpers.IsFileInShare(db, *share, file.ID, file) {
		return apierrors.New(http.StatusForbidden, apierrors.CodeShareFileNotInShare)
	}
	return nil
}
