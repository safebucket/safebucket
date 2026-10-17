package services

import (
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFileDownloadsRejectNonUploadedFiles(t *testing.T) {
	for _, status := range []models.FileStatus{
		models.FileStatusUploading,
		models.FileStatusDeleting,
		models.FileStatusDeleted,
		models.FileStatusRestoring,
	} {
		t.Run(string(status), func(t *testing.T) {
			service, file, version, user := setupFileDownloadTest(t)
			require.NoError(t, service.DB.Model(&file).Update("status", status).Error)
			service.Storage = nil

			for name, download := range map[string]func() (models.FileDownloadResponse, error){
				"current": func() (models.FileDownloadResponse, error) {
					return service.DownloadFile(zap.NewNop(), user, uuid.UUIDs{file.BucketID, file.ID},
						models.FileDownloadQuery{})
				},
				"version": func() (models.FileDownloadResponse, error) {
					return service.DownloadFileVersion(zap.NewNop(), user, uuid.UUIDs{file.BucketID, file.ID, version.ID},
						models.FileDownloadQuery{})
				},
			} {
				t.Run(name, func(t *testing.T) {
					_, err := download()
					var apiErr *apierrors.APIError
					require.ErrorAs(t, err, &apiErr)
					assert.Equal(t, http.StatusNotFound, apiErr.Status)
					assert.Equal(t, apierrors.CodeFileNotFound, apiErr.Code)
				})
			}
		})
	}
}

func TestFileDownloadsSelectObject(t *testing.T) {
	for _, test := range []struct {
		name            string
		currentVersion  bool
		explicitVersion bool
	}{
		{name: "legacy file"},
		{name: "current version", currentVersion: true},
		{name: "explicit older version", currentVersion: true, explicitVersion: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, file, version, user := setupFileDownloadTest(t)
			expectedID := file.ID
			if test.currentVersion {
				require.NoError(t, service.DB.Model(&file).Update("current_version_id", version.ID).Error)
				expectedID = version.ID
			}

			ids := uuid.UUIDs{file.BucketID, file.ID}
			if test.explicitVersion {
				older := models.FileVersion{FileID: file.ID, Version: 1, Size: 3, Status: models.FileStatusUploaded}
				require.NoError(t, service.DB.Create(&older).Error)
				ids = append(ids, older.ID)
				expectedID = older.ID
			}

			for _, context := range []string{"download", "preview"} {
				t.Run(context, func(t *testing.T) {
					store := &fileDownloadStorage{}
					service.Storage = store
					download := service.DownloadFile
					if test.explicitVersion {
						download = service.DownloadFileVersion
					}
					response, err := download(zap.NewNop(), user, ids, models.FileDownloadQuery{Context: context})
					require.NoError(t, err)
					assert.Equal(t, file.ID.String(), response.ID)
					assert.Equal(t, "buckets/"+file.BucketID.String()+"/"+expectedID.String(), response.URL)
					assert.Equal(t, file.Name, store.options.DownloadFilename)
					if context == "preview" {
						assert.Equal(t, "application/pdf", store.options.InlineContentType)
					} else {
						assert.Empty(t, store.options.InlineContentType)
					}
				})
			}
		})
	}
}

type fileDownloadStorage struct {
	storage.IStorage

	options storage.GetObjectOptions
}

func (s *fileDownloadStorage) PresignedGetObject(objectPath string, options storage.GetObjectOptions) (string, error) {
	s.options = options
	return objectPath, nil
}

type fileDownloadActivityLogger struct {
	activity.IActivityLogger
}

func (fileDownloadActivityLogger) Send(models.Activity) error { return nil }

type fileDownloadPublisher struct {
	messaging.IPublisher
}

func (fileDownloadPublisher) Publish(...*message.Message) error { return nil }

func setupFileDownloadTest(t *testing.T) (BucketFileService, models.File, models.FileVersion, models.UserClaims) {
	t.Helper()

	db := setupUserServiceTestDB(t)
	user := models.User{
		Email:        "download@example.com",
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
		Role:         models.RoleUser,
	}
	require.NoError(t, db.Create(&user).Error)
	bucket := models.Bucket{Name: "downloads", CreatedBy: user.ID}
	require.NoError(t, db.Create(&bucket).Error)
	file := models.File{
		Name:      "report.pdf",
		Extension: "pdf",
		BucketID:  bucket.ID,
		Status:    models.FileStatusUploaded,
		Size:      5,
	}
	require.NoError(t, db.Create(&file).Error)
	version := models.FileVersion{FileID: file.ID, Version: 2, Size: file.Size, Status: models.FileStatusUploaded}
	require.NoError(t, db.Create(&version).Error)

	service := BucketFileService{
		DB:             db,
		ActivityLogger: fileDownloadActivityLogger{},
		Publisher:      fileDownloadPublisher{},
	}
	return service, file, version, models.UserClaims{UserID: user.ID, Email: user.Email}
}
