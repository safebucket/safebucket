//go:build integration

package sharing_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const qsSharePassword = "horsebatterystaple"

func TestQuickSharePublicAuth(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)

			owner := app.CreateUser(t, "qspublicauth@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, ownerToken, "qs-publicauth")

			openShare := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name: "open-share",
				Type: models.ShareTypeBucket,
			})
			secretShare := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name:     "secret-share",
				Type:     models.ShareTypeBucket,
				Password: qsSharePassword,
			})
			otherSecret := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
				Name:     "other-secret",
				Type:     models.ShareTypeBucket,
				Password: qsSharePassword,
			})

			t.Run("open share lists without auth", func(t *testing.T) {
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", openShare.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)
				assert.Equal(t, openShare.ID, resp.ID)
			})

			t.Run("password-protected share rejects missing cookie", func(t *testing.T) {
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", secretShare.ID), "", nil, nil)
				assert.Equal(t, http.StatusUnauthorized, status)
			})

			t.Run("wrong password rejected", func(t *testing.T) {
				status, cookie := app.AuthenticateShare(t, secretShare.ID.String(), "wrongpassword")
				assert.Equal(t, http.StatusUnauthorized, status)
				assert.Empty(t, cookie)
			})

			t.Run("auth on share without password is rejected", func(t *testing.T) {
				status, _ := app.AuthenticateShare(t, openShare.ID.String(), qsSharePassword)
				assert.Equal(t, http.StatusBadRequest, status)
			})

			t.Run("correct password unlocks listing", func(t *testing.T) {
				status, cookie := app.AuthenticateShare(t, secretShare.ID.String(), qsSharePassword)
				require.Equal(t, http.StatusOK, status)
				require.NotEmpty(t, cookie)

				var resp models.PublicShareResponse
				getStatus := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", secretShare.ID), cookie, nil, &resp)
				require.Equal(t, http.StatusOK, getStatus)
				assert.Equal(t, secretShare.ID, resp.ID)
			})

			t.Run("share cookie is bound to its share", func(t *testing.T) {
				status, cookieForOther := app.AuthenticateShare(t, otherSecret.ID.String(), qsSharePassword)
				require.Equal(t, http.StatusOK, status)
				require.NotEmpty(t, cookieForOther)

				crossStatus := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", secretShare.ID), cookieForOther, nil, nil)
				assert.Equal(t, http.StatusUnauthorized, crossStatus)
			})

			t.Run("unknown share returns 404", func(t *testing.T) {
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", uuid.New()), "", nil, nil)
				assert.Equal(t, http.StatusNotFound, status)
			})
		})
	}
}

func TestQuickShareExpiryAndViews(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)

			owner := app.CreateUser(t, "qsgates@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, ownerToken, "qs-gates")

			t.Run("expired share returns 410", func(t *testing.T) {
				future := time.Now().Add(1 * time.Hour)
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:      "expiring-share",
					Type:      models.ShareTypeBucket,
					ExpiresAt: &future,
				})
				app.BackdateShareExpiry(t, share.ID.String(), time.Now().Add(-1*time.Hour))

				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, nil)
				assert.Equal(t, http.StatusGone, status)
			})

			t.Run("max views enforced after limit reached", func(t *testing.T) {
				maxViews := 2
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:     "limited-share",
					Type:     models.ShareTypeBucket,
					MaxViews: &maxViews,
				})
				sharePath := fmt.Sprintf("/api/v1/shares/%s", share.ID)

				var first models.PublicShareResponse
				require.Equal(t, http.StatusOK, app.DoPublicShare(t, http.MethodGet, sharePath, "", nil, &first))
				assert.Equal(t, 1, first.CurrentViews)

				var second models.PublicShareResponse
				require.Equal(t, http.StatusOK, app.DoPublicShare(t, http.MethodGet, sharePath, "", nil, &second))
				assert.Equal(t, 2, second.CurrentViews)

				third := app.DoPublicShare(t, http.MethodGet, sharePath, "", nil, nil)
				assert.Equal(t, http.StatusForbidden, third)
			})
		})
	}
}

func TestQuickShareScopeFiltering(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)

			owner := app.CreateUser(t, "qsscopes@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, ownerToken, "qs-scopes")

			folder := app.CreateFolder(t, ownerToken, bucket.ID.String(), "primary")
			otherFolder := app.CreateFolder(t, ownerToken, bucket.ID.String(), "secondary")

			rootFile := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "root.txt")

			folderFileResp := app.UploadFileInto(t, ownerToken, bucket.ID.String(), &folder.ID, "in-primary.txt")
			otherFolderFileResp := app.UploadFileInto(t, ownerToken, bucket.ID.String(),
				&otherFolder.ID, "in-secondary.txt")

			rootUUID := uuid.MustParse(rootFile)

			t.Run("files scope returns only linked files", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:    "files-share",
					Type:    models.ShareTypeFiles,
					FileIDs: []uuid.UUID{rootUUID},
				})
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)
				require.Len(t, resp.Files, 1)
				assert.Equal(t, rootUUID, resp.Files[0].ID)
				assert.Empty(t, resp.Folders)
			})

			t.Run("folder scope returns only direct children", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name:     "folder-share",
					Type:     models.ShareTypeFolder,
					FolderID: &folder.ID,
				})
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)

				fileIDs := fileIDSet(resp.Files)
				assert.Contains(t, fileIDs, folderFileResp)
				assert.NotContains(t, fileIDs, rootFile, "root file is not in folder scope")
				assert.NotContains(t, fileIDs, otherFolderFileResp, "sibling folder file is out of scope")
			})

			t.Run("bucket scope returns everything in the bucket", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name: "bucket-share",
					Type: models.ShareTypeBucket,
				})
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)

				fileIDs := fileIDSet(resp.Files)
				assert.Contains(t, fileIDs, rootFile)
				assert.Contains(t, fileIDs, folderFileResp)
				assert.Contains(t, fileIDs, otherFolderFileResp)

				folderIDs := make(map[string]struct{}, len(resp.Folders))
				for _, f := range resp.Folders {
					folderIDs[f.ID.String()] = struct{}{}
				}
				assert.Contains(t, folderIDs, folder.ID.String())
				assert.Contains(t, folderIDs, otherFolder.ID.String())
			})

			t.Run("trashed files are filtered out", func(t *testing.T) {
				trashedID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "soon-trashed.txt")
				app.TrashFile(t, ownerToken, bucket.ID.String(), trashedID)

				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name: "trash-filter-share",
					Type: models.ShareTypeBucket,
				})
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)

				assert.NotContains(t, fileIDSet(resp.Files), trashedID, "trashed file must not appear")
			})

			t.Run("expired files are filtered out", func(t *testing.T) {
				expiringID := app.UploadTestFile(t, ownerToken, bucket.ID.String(), "soon-expired.txt")
				app.BackdateFileExpiry(t, expiringID, time.Now().Add(-1*time.Hour))

				share := app.CreateShare(t, ownerToken, bucket.ID.String(), models.ShareCreateBody{
					Name: "expiry-filter-share",
					Type: models.ShareTypeBucket,
				})
				var resp models.PublicShareResponse
				status := app.DoPublicShare(t, http.MethodGet,
					fmt.Sprintf("/api/v1/shares/%s", share.ID), "", nil, &resp)
				require.Equal(t, http.StatusOK, status)

				assert.NotContains(t, fileIDSet(resp.Files), expiringID, "expired file must not appear")
			})
		})
	}
}

func fileIDSet(files []models.File) map[string]struct{} {
	out := make(map[string]struct{}, len(files))
	for _, f := range files {
		out[f.ID.String()] = struct{}{}
	}
	return out
}
