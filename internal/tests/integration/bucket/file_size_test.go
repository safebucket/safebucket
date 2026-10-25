//go:build integration

package bucket_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSizeFollowsCurrentVersion(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "file-size-owner@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "file-size-bucket")
			bucketID := bucket.ID.String()

			fileID := app.UploadTestFile(t, token, bucketID, "report.txt")
			assert.Equal(t, 5, bucketFileSize(t, app, token, bucketID, fileID))

			t.Run("re-upload shows the new version size", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(
					t,
					http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/files", bucketID),
					token,
					models.FileUploadBody{Name: "report.txt", Size: 7},
					&transfer,
				)
				require.Equal(t, http.StatusCreated, status)

				app.PutPresigned(t, transfer, []byte("seven!!"))

				require.Equal(t, http.StatusNoContent,
					app.DoStatus(
						t,
						http.MethodPatch,
						fmt.Sprintf(
							"/api/v1/buckets/%s/files/%s/versions/%s",
							bucketID,
							transfer.ID,
							transfer.VersionID,
						),
						token,
						nil,
					))

				assert.Equal(t, 7, bucketFileSize(t, app, token, bucketID, fileID))
			})

			t.Run("restore shows the restored version size", func(t *testing.T) {
				var versions []models.FileVersionResponse
				require.Equal(t, http.StatusOK,
					app.Do(
						t,
						http.MethodGet,
						fmt.Sprintf(
							"/api/v1/buckets/%s/files/%s/versions",
							bucketID,
							fileID,
						),
						token,
						nil,
						&versions,
					))

				var first models.FileVersionResponse
				for _, version := range versions {
					if version.Version == 1 {
						first = version
					}
				}
				require.NotEmpty(t, first.ID)

				require.Equal(t, http.StatusNoContent,
					app.DoStatus(t, http.MethodPut,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, fileID), token,
						models.FileVersionRestoreBody{VersionID: first.ID}))

				assert.Equal(t, 5, bucketFileSize(t, app, token, bucketID, fileID))
			})

			t.Run("in-progress upload shows its declared size", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(
					t,
					http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/files", bucketID),
					token,
					models.FileUploadBody{Name: "pending.txt", Size: 11},
					&transfer,
				)
				require.Equal(t, http.StatusCreated, status)

				assert.Equal(t, 11, bucketFileSize(t, app, token, bucketID, transfer.ID))
			})

			t.Run("public share lists the current size", func(t *testing.T) {
				share := app.CreateShare(t, token, bucketID, models.ShareCreateBody{
					Name:     "file-size-share",
					Type:     models.ShareTypeBucket,
					Password: sharePassword33,
				})

				status, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
				require.Equal(t, http.StatusOK, status)
				require.NotEmpty(t, cookie)

				var listing models.PublicShareResponse
				require.Equal(
					t,
					http.StatusOK,
					app.DoPublicShare(
						t,
						http.MethodGet,
						fmt.Sprintf("/api/v1/shares/%s", share.ID),
						cookie,
						nil,
						&listing,
					),
				)
				assert.Equal(t, 5, fileByID(t, listing.Files, fileID).Size)
			})

			t.Run("trash lists the current size", func(t *testing.T) {
				app.TrashFile(t, token, bucketID, fileID)

				var trash models.Bucket
				require.Equal(
					t,
					http.StatusOK,
					app.Do(
						t,
						http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/trash", bucketID),
						token,
						nil,
						&trash,
					),
				)
				assert.Equal(t, 5, fileByID(t, trash.Files, fileID).Size)
			})
		})
	}
}

func bucketFileSize(t *testing.T, app *bootstrap.TestApp, token, bucketID, fileID string) int {
	t.Helper()

	var bucket models.Bucket
	require.Equal(t, http.StatusOK,
		app.Do(t, http.MethodGet, fmt.Sprintf("/api/v1/buckets/%s", bucketID), token, nil, &bucket))
	return fileByID(t, bucket.Files, fileID).Size
}
