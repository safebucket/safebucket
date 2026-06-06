//go:build integration

package sharing_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuickShareDownload(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)

			owner := app.CreateUser(t, "qsdownload@example.com")
			ownerToken := app.LoginAs(t, owner.Email)

			bucket := app.CreateBucket(t, ownerToken, "qs-download-primary")
			otherBucket := app.CreateBucket(t, ownerToken, "qs-download-other")

			fileID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "shared.txt")
			unlinkedID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "unlinked.txt")
			otherBucketFileID := app.UploadTestFile(t, ownerToken, otherBucket.ID.String(), "other.txt")

			bucketShare := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name: "bucket-download-share",
				Type: models.ShareTypeBucket,
			})

			filesShare := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name:    "files-download-share",
				Type:    models.ShareTypeFiles,
				FileIDs: []uuid.UUID{uuid.MustParse(fileID)},
			})

			t.Run("happy path returns a working presigned URL", func(t *testing.T) {
				var transfer models.FileTransferResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s/files/%s", bucketShare.ID, fileID),
					"", nil, &transfer)
				require.Equal(t, http.StatusOK, status)
				require.NotEmpty(t, transfer.URL, "response must include a presigned URL")

				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, transfer.URL, nil)
				require.NoError(t, err)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode, "presigned URL must serve the object")
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "test!", string(body), "object body must match what UploadTestFile wrote")
			})

			t.Run("files scope rejects file not linked to share", func(t *testing.T) {
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s/files/%s", filesShare.ID, unlinkedID),
					"", nil, nil)
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects file from another bucket", func(t *testing.T) {
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s/files/%s", bucketShare.ID, otherBucketFileID),
					"", nil, nil)
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects unknown file id", func(t *testing.T) {
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s/files/%s", bucketShare.ID, uuid.New()),
					"", nil, nil)
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("activity log records share download", func(t *testing.T) {
				activityShare := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name: "activity-share",
					Type: models.ShareTypeBucket,
				})

				var transfer models.FileTransferResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s/files/%s", activityShare.ID, fileID),
					"", nil, &transfer)
				require.Equal(t, http.StatusOK, status)

				results, err := app.Activity.Search(map[string][]string{
					"share_id": {activityShare.ID.String()},
				}, time.Now().AddDate(0, 0, -30), time.Now(), 100)
				require.NoError(t, err)
				require.NotEmpty(t, results, "share download must produce an activity entry")

				var matched bool
				for _, r := range results {
					if r["share_id"] == activityShare.ID.String() &&
						r["file_id"] == fileID &&
						r["action"] == "download" {
						matched = true
						break
					}
				}
				assert.True(t, matched, "expected download activity for share+file (results=%v)", results)
			})
		})
	}
}
