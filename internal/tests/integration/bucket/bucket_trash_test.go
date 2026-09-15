//go:build integration

package bucket_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketTrash(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "trash-owner@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "trash-bucket")
			bucketID := bucket.ID.String()
			trashPath := fmt.Sprintf("/api/v1/buckets/%s/trash", bucketID)

			t.Run("trashes duplicate selections and folder contents", func(t *testing.T) {
				parent := app.CreateFolder(t, token, bucketID, "parent")
				child := createFolder(t, app, token, bucketID, "child", &parent.ID)
				rootFileID := app.UploadTestFile(t, token, bucketID, "root.txt")
				childFileID := app.UploadFileInto(t, token, bucketID, &child.ID, "child.txt")

				status := app.DoStatus(t, http.MethodPost, trashPath, token, models.BucketTrashBody{
					FolderIDs: []uuid.UUID{parent.ID, parent.ID},
					FileIDs:   []uuid.UUID{uuid.MustParse(rootFileID), uuid.MustParse(rootFileID)},
				})
				require.Equal(t, http.StatusAccepted, status)

				waitForTrashed(t, app, []string{parent.ID.String(), child.ID.String()}, []string{rootFileID, childFileID})

				var trash models.Bucket
				app.Eventually(t, func() bool {
					return app.Do(t, http.MethodGet, trashPath, token, nil, &trash) == http.StatusOK &&
						len(trash.Folders) == 2 && len(trash.Files) == 2
				}, "trashed items should be listed")

				assert.Equal(t, "/", folderByID(t, trash.Folders, parent.ID).OriginalPath)
				assert.Equal(t, "/parent", folderByID(t, trash.Folders, child.ID).OriginalPath)
				assert.Equal(t, "/", fileByID(t, trash.Files, rootFileID).OriginalPath)
				assert.Equal(t, "/parent/child", fileByID(t, trash.Files, childFileID).OriginalPath)
			})

			t.Run("rejects invalid selections without changing valid items", func(t *testing.T) {
				validFileID := app.UploadTestFile(t, token, bucketID, "valid.txt")
				missingFolderID := uuid.New()

				status, codes := app.DoExpectError(t, http.MethodPost, trashPath, token, models.BucketTrashBody{
					FileIDs:   []uuid.UUID{uuid.MustParse(validFileID)},
					FolderIDs: []uuid.UUID{missingFolderID},
				})
				require.Equal(t, http.StatusNotFound, status)
				assert.Contains(t, codes, apierrors.CodeFolderNotFound)

				var validFile models.File
				require.NoError(t, app.DB().First(&validFile, "id = ?", validFileID).Error)
				assert.Equal(t, models.FileStatusUploaded, validFile.Status)
				assert.False(t, validFile.DeletedAt.Valid)

				status, codes = app.DoExpectError(t, http.MethodPost, trashPath, token, models.BucketTrashBody{})
				assert.Equal(t, http.StatusBadRequest, status)
				assert.Contains(t, codes, apierrors.CodeInvalidValue)
			})

			t.Run("rejects non-trashable, expired, and already trashed files", func(t *testing.T) {
				var uploading models.FileUploadResponse
				require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "uploading.txt", Size: 5}, &uploading))

				status, codes := app.DoExpectError(t, http.MethodPost, trashPath, token,
					models.BucketTrashBody{FileIDs: []uuid.UUID{uuid.MustParse(uploading.ID)}})
				assert.Equal(t, http.StatusConflict, status)
				assert.Contains(t, codes, apierrors.CodeItemNotTrashable)

				expiredID := app.UploadTestFile(t, token, bucketID, "expired.txt")
				app.BackdateFileExpiry(t, expiredID, time.Now().Add(-time.Minute))
				status, codes = app.DoExpectError(t, http.MethodPost, trashPath, token,
					models.BucketTrashBody{FileIDs: []uuid.UUID{uuid.MustParse(expiredID)}})
				assert.Equal(t, http.StatusForbidden, status)
				assert.Contains(t, codes, apierrors.CodeFileExpired)

				trashedID := app.UploadTestFile(t, token, bucketID, "already-trashed.txt")
				app.TrashFile(t, token, bucketID, trashedID)
				status, codes = app.DoExpectError(t, http.MethodPost, trashPath, token,
					models.BucketTrashBody{FileIDs: []uuid.UUID{uuid.MustParse(trashedID)}})
				assert.Equal(t, http.StatusConflict, status)
				assert.Contains(t, codes, apierrors.CodeItemAlreadyTrashed)
			})
		})
	}
}

func createFolder(
	t *testing.T,
	app *bootstrap.TestApp,
	token, bucketID, name string,
	parentID *uuid.UUID,
) models.Folder {
	t.Helper()

	var folder models.Folder
	status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/folders", bucketID), token,
		models.FolderCreateBody{Name: name, FolderID: parentID}, &folder)
	require.Equal(t, http.StatusCreated, status)
	return folder
}

func waitForTrashed(t *testing.T, app *bootstrap.TestApp, folderIDs, fileIDs []string) {
	t.Helper()

	app.Eventually(t, func() bool {
		for _, id := range folderIDs {
			var count int64
			if app.DB().Unscoped().Model(&models.Folder{}).
				Where("id = ? AND status = ? AND deleted_at IS NOT NULL", id, models.FolderStatusDeleted).
				Count(&count).Error != nil || count != 1 {
				return false
			}
		}
		for _, id := range fileIDs {
			var count int64
			if app.DB().Unscoped().Model(&models.File{}).
				Where("id = ? AND status = ? AND deleted_at IS NOT NULL", id, models.FileStatusDeleted).
				Count(&count).Error != nil || count != 1 {
				return false
			}
		}
		return true
	}, "all selected items and folder contents should be trashed")
}

func folderByID(t *testing.T, folders []models.Folder, id uuid.UUID) models.Folder {
	t.Helper()
	for _, folder := range folders {
		if folder.ID == id {
			return folder
		}
	}
	t.Fatalf("folder %s not found", id)
	return models.Folder{}
}

func fileByID(t *testing.T, files []models.File, id string) models.File {
	t.Helper()
	for _, file := range files {
		if file.ID.String() == id {
			return file
		}
	}
	t.Fatalf("file %s not found", id)
	return models.File{}
}
