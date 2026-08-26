//go:build integration

package bucket_test

import (
	"fmt"
	"net/http"
	"path"
	"testing"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const eventuallyTimeout = 10 * time.Second
const eventuallyTick = 100 * time.Millisecond

type trashFixture struct {
	app    *bootstrap.TestApp
	token  string
	bucket models.Bucket
}

func newTrashFixture(t *testing.T, app *bootstrap.TestApp, email, bucketName string) *trashFixture {
	t.Helper()

	user := app.CreateUser(t, email)
	token := app.LoginAs(t, user.Email)
	bucket := app.CreateBucket(t, token, bucketName)
	return &trashFixture{app: app, token: token, bucket: bucket}
}

func (f *trashFixture) listItems(
	t *testing.T, status string,
) (map[string]models.File, map[string]models.Folder) {
	t.Helper()

	var bucket models.Bucket
	reqPath := fmt.Sprintf("/api/v1/buckets/%s?status=%s", f.bucket.ID, status)
	require.Equal(t, http.StatusOK,
		f.app.Do(t, http.MethodGet, reqPath, f.token, nil, &bucket),
		"list bucket items with status %s", status)

	files := make(map[string]models.File, len(bucket.Files))
	for _, file := range bucket.Files {
		files[file.ID.String()] = file
	}
	folders := make(map[string]models.Folder, len(bucket.Folders))
	for _, folder := range bucket.Folders {
		folders[folder.ID.String()] = folder
	}
	return files, folders
}

func (f *trashFixture) patchItemStatus(t *testing.T, kind, id, status string) (int, []string) {
	t.Helper()
	reqPath := fmt.Sprintf("/api/v1/buckets/%s/%s/%s", f.bucket.ID, kind, id)

	var body any
	if kind == "folders" {
		body = models.FolderPatchBody{Status: models.FolderStatus(status)}
	} else {
		body = models.FilePatchBody{Status: status}
	}
	return f.app.DoExpectError(t, http.MethodPatch, reqPath, f.token, body)
}

func (f *trashFixture) bulkTrash(t *testing.T, folderIDs, fileIDs []uuid.UUID) int {
	t.Helper()
	return f.app.DoStatus(t, http.MethodPost,
		fmt.Sprintf("/api/v1/buckets/%s/files/trash", f.bucket.ID), f.token,
		models.BulkTrashBody{FolderIDs: folderIDs, FileIDs: fileIDs})
}

func (f *trashFixture) mustTrash(t *testing.T, kind, id string) {
	t.Helper()

	var status int
	if kind == "folders" {
		status = f.bulkTrash(t, []uuid.UUID{mustUUID(t, id)}, nil)
	} else {
		status = f.bulkTrash(t, nil, []uuid.UUID{mustUUID(t, id)})
	}
	require.Equal(t, http.StatusNoContent, status, "trash %s %s", kind, id)

	assert.Eventually(t, func() bool {
		trashedFiles, trashedFolders := f.listItems(t, "deleted")
		if kind == "folders" {
			_, ok := trashedFolders[id]
			return ok
		}
		_, ok := trashedFiles[id]
		return ok
	}, eventuallyTimeout, eventuallyTick, "%s %s should be trashed", kind, id)
}

func (f *trashFixture) createFolderIn(
	t *testing.T, name string, parentID *uuid.UUID,
) models.Folder {
	t.Helper()

	var folder models.Folder
	status := f.app.Do(t, http.MethodPost,
		fmt.Sprintf("/api/v1/buckets/%s/folders", f.bucket.ID), f.token,
		models.FolderCreateBody{Name: name, FolderID: parentID}, &folder)
	require.Equal(t, http.StatusCreated, status, "create folder %s", name)
	return folder
}

func TestTrashFileLifecycle(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			f := newTrashFixture(t, app, "lifecycle-owner@example.com", "lifecycle-bucket")

			t.Run("trash hides the file and blocks downloads", func(t *testing.T) {
				fileID := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "lifecycle.txt")

				status := f.bulkTrash(t, nil, []uuid.UUID{mustUUID(t, fileID)})
				require.Equal(t, http.StatusNoContent, status)

				assert.Eventually(t, func() bool {
					trashedFiles, _ := f.listItems(t, "deleted")
					if _, ok := trashedFiles[fileID]; !ok {
						return false
					}
					liveFiles, _ := f.listItems(t, "uploaded")
					_, stillLive := liveFiles[fileID]
					return !stillLive
				}, eventuallyTimeout, eventuallyTick, "file should move from list to trash")

				dlStatus, dlCodes := f.app.DoExpectError(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s/files/%s/url", f.bucket.ID, fileID), f.token, nil)
				assert.Equal(t, http.StatusNotFound, dlStatus)
				assert.Contains(t, dlCodes, apierrors.CodeFileNotFound)
			})

			t.Run("re-trashing an already trashed file is a no-op", func(t *testing.T) {
				fileID := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "double.txt")
				f.mustTrash(t, "files", fileID)

				status := f.bulkTrash(t, nil, []uuid.UUID{mustUUID(t, fileID)})
				require.Equal(t, http.StatusNoContent, status)

				trashedFiles, _ := f.listItems(t, "deleted")
				assert.Contains(t, trashedFiles, fileID)
				liveFiles, _ := f.listItems(t, "uploaded")
				assert.NotContains(t, liveFiles, fileID)
			})

			t.Run("restore brings the file back", func(t *testing.T) {
				fileID := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "restore-me.txt")
				f.mustTrash(t, "files", fileID)

				status, _ := f.patchItemStatus(t, "files", fileID, string(models.FileStatusUploaded))
				require.Equal(t, http.StatusNoContent, status)

				var dl models.FileDownloadResponse
				assert.Eventually(t, func() bool {
					liveFiles, _ := f.listItems(t, "uploaded")
					if _, ok := liveFiles[fileID]; !ok {
						return false
					}
					downloadStatus := f.app.Do(t, http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s/url", f.bucket.ID, fileID),
						f.token, nil, &dl)
					return downloadStatus == http.StatusOK && dl.URL != ""
				}, eventuallyTimeout, eventuallyTick, "restored file should be listed and downloadable")
			})

			t.Run("restore conflicts when the name is taken", func(t *testing.T) {
				fileID := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "name-conflict.txt")
				f.mustTrash(t, "files", fileID)

				app.UploadTestFile(t, f.token, f.bucket.ID.String(), "name-conflict.txt")

				status, codes := f.patchItemStatus(t, "files", fileID, string(models.FileStatusUploaded))
				assert.Equal(t, http.StatusConflict, status)
				assert.Contains(t, codes, apierrors.CodeFileNameConflict)
			})
		})
	}
}

func TestFolderTrashCascade(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			f := newTrashFixture(t, app, "cascade-owner@example.com", "cascade-bucket")

			parent := f.createFolderIn(t, "cascade-parent", nil)
			child := f.createFolderIn(t, "cascade-child", &parent.ID)
			rootFile := app.UploadFileInto(t, f.token, f.bucket.ID.String(), &parent.ID, "in-parent.txt")
			nestedFile := app.UploadFileInto(t, f.token, f.bucket.ID.String(), &child.ID, "in-child.txt")
			looseFile := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "untouched.txt")

			status := f.bulkTrash(t, []uuid.UUID{parent.ID}, nil)
			require.Equal(t, http.StatusNoContent, status)

			t.Run("children trashed asynchronously", func(t *testing.T) {
				assert.Eventually(t, func() bool {
					trashedFiles, trashedFolders := f.listItems(t, "deleted")
					_, childTrashed := trashedFolders[child.ID.String()]
					_, rootFileTrashed := trashedFiles[rootFile]
					_, nestedFileTrashed := trashedFiles[nestedFile]
					return childTrashed && rootFileTrashed && nestedFileTrashed
				}, eventuallyTimeout, eventuallyTick, "subtree items should be trashed")
			})

			t.Run("items outside the subtree stay live", func(t *testing.T) {
				liveFiles, _ := f.listItems(t, "uploaded")
				assert.Contains(t, liveFileIDs(liveFiles), looseFile)
			})

			t.Run("restoring the root recovers the whole subtree", func(t *testing.T) {
				status, _ := f.patchItemStatus(t, "folders", parent.ID.String(), string(models.FolderStatusCreated))
				require.Equal(t, http.StatusNoContent, status)

				assert.Eventually(t, func() bool {
					liveFiles, liveFolders := f.listItems(t, "uploaded")
					_, parentBack := liveFolders[parent.ID.String()]
					_, childBack := liveFolders[child.ID.String()]
					_, rootFileBack := liveFiles[rootFile]
					_, nestedFileBack := liveFiles[nestedFile]
					return parentBack && childBack && rootFileBack && nestedFileBack
				}, eventuallyTimeout, eventuallyTick, "subtree should return to the list")
			})
		})
	}
}

func liveFileIDs(files map[string]models.File) []string {
	ids := make([]string, 0, len(files))
	for id := range files {
		ids = append(ids, id)
	}
	return ids
}

func TestBulkTrash(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.WithLocalSharing(bootstrap.LoadScenario(t, scenario), true)
			app := bootstrap.BootTestApp(t, cfg)
			f := newTrashFixture(t, app, "bulk-owner@example.com", "bulk-bucket")
			bulkPath := fmt.Sprintf("/api/v1/buckets/%s/files/trash", f.bucket.ID)

			fileA := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "bulk-a.txt")
			fileB := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "bulk-b.txt")
			folderA := f.createFolderIn(t, "bulk-folder-a", nil)
			folderB := f.createFolderIn(t, "bulk-folder-b", nil)
			nestedFile := app.UploadFileInto(t, f.token, f.bucket.ID.String(), &folderB.ID, "nested.txt")

			t.Run("mixed selection is accepted and processed", func(t *testing.T) {
				body := models.BulkTrashBody{
					FolderIDs: []uuid.UUID{folderA.ID, folderB.ID},
					FileIDs:   []uuid.UUID{mustUUID(t, fileA), mustUUID(t, fileB), mustUUID(t, fileB)},
				}

				status := app.DoStatus(t, http.MethodPost, bulkPath, f.token, body)
				require.Equal(t, http.StatusNoContent, status)

				assert.Eventually(t, func() bool {
					trashedFiles, trashedFolders := f.listItems(t, "deleted")
					_, folderATrashed := trashedFolders[folderA.ID.String()]
					_, folderBTrashed := trashedFolders[folderB.ID.String()]
					_, fileATrashed := trashedFiles[fileA]
					_, fileBTrashed := trashedFiles[fileB]
					_, nestedTrashed := trashedFiles[nestedFile]
					return folderATrashed && folderBTrashed &&
						fileATrashed && fileBTrashed && nestedTrashed
				}, eventuallyTimeout, eventuallyTick,
					"all selected items plus folder contents should be trashed")
			})

			t.Run("unknown ids are accepted and skipped silently", func(t *testing.T) {
				body := models.BulkTrashBody{
					FileIDs: []uuid.UUID{uuid.New()},
				}

				status := app.DoStatus(t, http.MethodPost, bulkPath, f.token, body)
				assert.Equal(t, http.StatusNoContent, status)
			})

			t.Run("empty batch is rejected", func(t *testing.T) {
				status, codes := app.DoExpectError(t, http.MethodPost, bulkPath, f.token,
					models.BulkTrashBody{})
				assert.Equal(t, http.StatusBadRequest, status)
				assert.NotEmpty(t, codes)
			})

			t.Run("oversized batch is rejected", func(t *testing.T) {
				tooMany := make([]uuid.UUID, 1001)
				for i := range tooMany {
					tooMany[i] = uuid.New()
				}

				status, _ := app.DoExpectError(t, http.MethodPost, bulkPath, f.token,
					models.BulkTrashBody{FileIDs: tooMany})
				assert.Equal(t, http.StatusBadRequest, status)
			})

			t.Run("viewer cannot bulk trash", func(t *testing.T) {
				viewer := app.CreateUser(t, "bulk-viewer@example.com")
				viewerToken := app.LoginAs(t, viewer.Email)
				app.AddMembers(t, f.token, f.bucket.ID.String(),
					[]models.BucketMemberBody{{Email: viewer.Email, Group: models.GroupViewer}})

				status, _ := app.DoExpectError(t, http.MethodPost, bulkPath, viewerToken,
					models.BulkTrashBody{FileIDs: []uuid.UUID{mustUUID(t, fileA)}})
				assert.Equal(t, http.StatusForbidden, status)
			})
		})
	}
}

func TestPurgeFile(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			f := newTrashFixture(t, app, "purge-owner@example.com", "purge-bucket")

			fileID := app.UploadTestFile(t, f.token, f.bucket.ID.String(), "purge-me.txt")
			objectPath := path.Join("buckets", f.bucket.ID.String(), fileID)

			_, err := f.app.Storage.StatObject(objectPath)
			require.NoError(t, err, "object should exist before purge")

			f.mustTrash(t, "files", fileID)

			status, _ := f.app.DoExpectError(t, http.MethodDelete,
				fmt.Sprintf("/api/v1/buckets/%s/files/%s", f.bucket.ID, fileID), f.token, nil)
			require.Equal(t, http.StatusNoContent, status)

			assert.Eventually(t, func() bool {
				_, statErr := f.app.Storage.StatObject(objectPath)
				return statErr != nil
			}, eventuallyTimeout, eventuallyTick, "object should be removed from storage")

			trashedFiles, _ := f.listItems(t, "deleted")
			assert.NotContains(t, trashedFiles, fileID)
			liveFiles, _ := f.listItems(t, "uploaded")
			assert.NotContains(t, liveFiles, fileID)
		})
	}
}

func mustUUID(t *testing.T, id string) uuid.UUID {
	t.Helper()
	parsed, err := uuid.Parse(id)
	require.NoError(t, err)
	return parsed
}
