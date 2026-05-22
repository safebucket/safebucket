//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuickShareHarness(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			owner := app.CreateUser(t, "qsharness@example.com")
			token := app.LoginAs(t, owner.Email)

			bucket := app.CreateBucket(t, token, "qs-harness")
			folder := app.CreateFolder(t, token, bucket.ID.String(), "qs-folder")

			share := app.CreateShare(t, token, bucket.ID.String(), models.ShareCreateBody{
				Name: "harness-share",
				Type: models.ShareTypeBucket,
			})

			require.NotEqual(t, "", share.ID.String(), "share ID must be populated")
			assert.Equal(t, models.ShareTypeBucket, share.Type)
			assert.Equal(t, bucket.ID, share.BucketID)
			assert.False(t, share.PasswordProtected, "no password supplied")
			assert.False(t, share.AllowUpload)
			assert.Equal(t, 0, share.CurrentViews)
			assert.Equal(t, 0, share.CurrentUploads)

			assert.Equal(t, bucket.ID, folder.BucketID, "folder belongs to bucket")

			var publicView models.PublicShareResponse
			status := app.doPublicShare(t, http.MethodGet,
				fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &publicView)
			require.Equal(t, http.StatusOK, status, "open share must be publicly listable")
			assert.Equal(t, share.ID, publicView.ID)
			assert.Equal(t, 1, publicView.CurrentViews, "first public view increments counter")
		})
	}
}

func TestQuickShareCreate(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := LoadScenario(t, scenario)
			cfg = WithLocalSharing(cfg, true)
			app := BootTestApp(t, cfg)

			owner := app.CreateUser(t, "qsowner@example.com")
			contrib := app.CreateUser(t, "qscontrib@example.com")
			viewer := app.CreateUser(t, "qsviewer@example.com")
			outsider := app.CreateUser(t, "qsoutsider@example.com")

			ownerToken := app.LoginAs(t, owner.Email)
			contribToken := app.LoginAs(t, contrib.Email)
			viewerToken := app.LoginAs(t, viewer.Email)
			outsiderToken := app.LoginAs(t, outsider.Email)

			bucket := app.CreateBucket(t, ownerToken, "qs-primary")
			otherBucket := app.CreateBucket(t, ownerToken, "qs-other")

			app.AddMembers(t, ownerToken, bucket.ID.String(), []models.BucketMemberBody{
				{Email: contrib.Email, Group: models.GroupContributor},
				{Email: viewer.Email, Group: models.GroupViewer},
			})

			folder := app.CreateFolder(t, ownerToken, bucket.ID.String(), "primary-folder")
			otherFolder := app.CreateFolder(t, ownerToken, otherBucket.ID.String(), "other-folder")
			fileID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "primary.txt")
			otherFileID := app.UploadTestFile(t, ownerToken, otherBucket.ID.String(), "other.txt")

			postShare := func(token string, body models.ShareCreateBody) int {
				return app.DoStatus(t, http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/shares", bucket.ID), token, body)
			}

			t.Run("bucket scope happy path", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name: "bucket-share",
					Type: models.ShareTypeBucket,
				})
				assert.Equal(t, models.ShareTypeBucket, share.Type)
				assert.Equal(t, bucket.ID, share.BucketID)
				assert.False(t, share.PasswordProtected)
				assert.False(t, share.AllowUpload)
				assert.Nil(t, share.FolderID)
			})

			t.Run("folder scope happy path", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:     "folder-share",
					Type:     models.ShareTypeFolder,
					FolderID: &folder.ID,
				})
				assert.Equal(t, models.ShareTypeFolder, share.Type)
				require.NotNil(t, share.FolderID)
				assert.Equal(t, folder.ID, *share.FolderID)
			})

			t.Run("files scope happy path", func(t *testing.T) {
				secondID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "second.txt")
				secondUUID := uuid.MustParse(secondID)
				firstUUID := uuid.MustParse(fileID)

				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:    "files-share",
					Type:    models.ShareTypeFiles,
					FileIDs: []uuid.UUID{firstUUID, secondUUID},
				})
				assert.Equal(t, models.ShareTypeFiles, share.Type)
				assert.Len(t, share.Files, 2)
			})

			t.Run("rejects unknown file id", func(t *testing.T) {
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:    "ghost-file-share",
					Type:    models.ShareTypeFiles,
					FileIDs: []uuid.UUID{uuid.New()},
				})
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects file from another bucket", func(t *testing.T) {
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:    "cross-bucket-files-share",
					Type:    models.ShareTypeFiles,
					FileIDs: []uuid.UUID{uuid.MustParse(otherFileID)},
				})
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects unknown folder id", func(t *testing.T) {
				unknown := uuid.New()
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:     "ghost-folder-share",
					Type:     models.ShareTypeFolder,
					FolderID: &unknown,
				})
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects folder from another bucket", func(t *testing.T) {
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:     "cross-bucket-folder-share",
					Type:     models.ShareTypeFolder,
					FolderID: &otherFolder.ID,
				})
				assert.Equal(t, http.StatusNotFound, status)
			})

			t.Run("rejects past expiry", func(t *testing.T) {
				past := time.Now().Add(-1 * time.Hour)
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:      "expired-share",
					Type:      models.ShareTypeBucket,
					ExpiresAt: &past,
				})
				assert.Equal(t, http.StatusBadRequest, status)
			})

			t.Run("rejects allow_upload with files scope", func(t *testing.T) {
				firstUUID := uuid.MustParse(fileID)
				status := postShare(ownerToken, models.ShareCreateBody{
					Name:        "files-upload-share",
					Type:        models.ShareTypeFiles,
					FileIDs:     []uuid.UUID{firstUUID},
					AllowUpload: true,
				})
				assert.Equal(t, http.StatusBadRequest, status)
			})

			t.Run("password is hashed and flag returned", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:     "secret-share",
					Type:     models.ShareTypeBucket,
					Password: "horsebatterystaple",
				})
				assert.True(t, share.PasswordProtected, "response must signal password protection")
				assert.Empty(t, share.HashedPassword, "hash must never leak via JSON")

				var page models.Page[models.Share]
				status := app.Do(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s/shares", bucket.ID), ownerToken, nil, &page)
				require.Equal(t, http.StatusOK, status)

				var listed *models.Share
				for i := range page.Data {
					if page.Data[i].ID == share.ID {
						listed = &page.Data[i]
						break
					}
				}
				require.NotNil(t, listed, "share must appear in list")
				assert.True(t, listed.PasswordProtected, "list must preserve password flag")
				assert.Empty(t, listed.HashedPassword, "list must not expose hash")
			})

			t.Run("contributor cannot create share", func(t *testing.T) {
				status := postShare(contribToken, models.ShareCreateBody{
					Name: "contrib-share",
					Type: models.ShareTypeBucket,
				})
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("viewer cannot create share", func(t *testing.T) {
				status := postShare(viewerToken, models.ShareCreateBody{
					Name: "viewer-share",
					Type: models.ShareTypeBucket,
				})
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("non-member cannot create share", func(t *testing.T) {
				status := postShare(outsiderToken, models.ShareCreateBody{
					Name: "outsider-share",
					Type: models.ShareTypeBucket,
				})
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("non-owner cannot list shares", func(t *testing.T) {
				listPath := fmt.Sprintf("/api/v1/buckets/%s/shares", bucket.ID)
				assert.Equal(t, http.StatusForbidden, app.DoStatus(t, http.MethodGet, listPath, contribToken, nil))
				assert.Equal(t, http.StatusForbidden, app.DoStatus(t, http.MethodGet, listPath, viewerToken, nil))
				assert.Equal(t, http.StatusForbidden, app.DoStatus(t, http.MethodGet, listPath, outsiderToken, nil))
			})
		})
	}
}

func TestQuickShareListAndDelete(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := LoadScenario(t, scenario)
			cfg = WithLocalSharing(cfg, true)
			app := BootTestApp(t, cfg)

			owner := app.CreateUser(t, "qsdelowner@example.com")
			contrib := app.CreateUser(t, "qsdelcontrib@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			contribToken := app.LoginAs(t, contrib.Email)

			bucket := app.CreateBucket(t, ownerToken, "qs-del-bucket")
			app.AddMembers(t, ownerToken, bucket.ID.String(), []models.BucketMemberBody{
				{Email: contrib.Email, Group: models.GroupContributor},
			})

			first := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name: "first-share",
				Type: models.ShareTypeBucket,
			})
			time.Sleep(50 * time.Millisecond)
			second := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name: "second-share",
				Type: models.ShareTypeBucket,
			})

			t.Run("list returns newest first", func(t *testing.T) {
				var page models.Page[models.Share]
				status := app.Do(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s/shares", bucket.ID), ownerToken, nil, &page)
				require.Equal(t, http.StatusOK, status)
				shares := page.Data
				require.Len(t, shares, 2)
				assert.Equal(t, second.ID, shares[0].ID, "newest share comes first")
				assert.Equal(t, first.ID, shares[1].ID)
			})

			t.Run("contributor cannot delete share", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodDelete,
					fmt.Sprintf("/api/v1/buckets/%s/shares/%s", bucket.ID, first.ID),
					contribToken, nil)
				assert.Equal(t, http.StatusForbidden, status)
			})

			t.Run("owner deletes share", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodDelete,
					fmt.Sprintf("/api/v1/buckets/%s/shares/%s", bucket.ID, first.ID),
					ownerToken, nil)
				require.Equal(t, http.StatusNoContent, status)

				var page models.Page[models.Share]
				status = app.Do(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s/shares", bucket.ID), ownerToken, nil, &page)
				require.Equal(t, http.StatusOK, status)
				assert.Len(t, page.Data, 1)
				assert.Equal(t, second.ID, page.Data[0].ID)
			})

			t.Run("delete unknown share returns 404", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodDelete,
					fmt.Sprintf("/api/v1/buckets/%s/shares/%s", bucket.ID, uuid.New()),
					ownerToken, nil)
				assert.Equal(t, http.StatusNotFound, status)
			})
		})
	}
}
