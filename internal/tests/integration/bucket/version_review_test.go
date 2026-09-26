//go:build integration

package bucket_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelledUploadIsHiddenAndLateWriteIsCleaned(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "cancel-version@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "cancel")
			bucketPath := fmt.Sprintf("/api/v1/buckets/%s", bucket.ID)
			var transfer models.FileUploadResponse
			require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost, bucketPath+"/files", token,
				models.FileUploadBody{Name: "cancel.txt", Size: 4}, &transfer))
			var listing models.Bucket
			require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, bucketPath, token, nil, &listing))
			require.Len(t, listing.Files, 1)
			versionPath := bucketPath + "/files/" + transfer.ID + "/versions/" + transfer.VersionID
			require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodDelete, versionPath, token, nil))
			require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, bucketPath, token, nil, &listing))
			assert.Empty(t, listing.Files)
			app.PutPresigned(t, transfer, []byte("late"))
			require.Equal(t, http.StatusConflict, app.DoStatus(t, http.MethodPatch, versionPath, token, nil))
			require.NoError(t, app.DB().Model(&models.FileVersion{}).Where("id = ?", transfer.VersionID).
				Update("cleanup_after", time.Now().Add(-time.Minute)).Error)
			manager := fileversions.Manager{DB: app.DB(), Storage: app.Storage, Cache: app.Cache}
			require.NoError(
				t,
				manager.Cleanup(
					bucket.ID,
					uuid.MustParse(transfer.ID),
					uuid.MustParse(transfer.VersionID),
				),
			)
			_, err := app.Storage.StatObject("buckets/" + bucket.ID.String() + "/" + transfer.VersionID)
			require.Error(t, err)
		})
	}
}

func TestConcurrentRestoreFileNames(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "restore-race@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "restore")
			bucketID := bucket.ID.String()
			base := fmt.Sprintf("/api/v1/buckets/%s/files", bucketID)
			first := app.UploadTestFile(t, token, bucketID, "same.txt")
			app.TrashFile(t, token, bucketID, first)
			waitForTrashed(t, app, nil, []string{first})
			second := app.UploadTestFile(t, token, bucketID, "same.txt")
			app.TrashFile(t, token, bucketID, second)
			waitForTrashed(t, app, nil, []string{second})
			start := make(chan struct{})
			results := make(chan int, 2)
			for _, id := range []string{first, second} {
				go func() {
					<-start
					results <- app.DoStatus(t, http.MethodPatch, base+"/"+id, token, models.FilePatchBody{Status: "uploaded"})
				}()
			}
			close(start)
			assert.ElementsMatch(t, []int{http.StatusNoContent, http.StatusConflict}, []int{<-results, <-results})
			var live []models.File
			require.NoError(t, app.DB().Where("bucket_id = ? AND name = ?", bucket.ID, "same.txt").Find(&live).Error)
			require.Len(t, live, 1)
			liveID := live[0].ID.String()
			app.TrashFile(t, token, bucketID, liveID)
			waitForTrashed(t, app, nil, []string{liveID})
			start = make(chan struct{})
			go func() {
				<-start
				results <- app.DoStatus(t, http.MethodPatch, base+"/"+liveID, token, models.FilePatchBody{Status: "uploaded"})
			}()
			go func() {
				<-start
				results <- app.DoStatus(t, http.MethodPost, base, token, models.FileUploadBody{Name: "same.txt", Size: 4})
			}()
			close(start)
			for range 2 {
				assert.Contains(t, []int{http.StatusCreated, http.StatusNoContent, http.StatusConflict}, <-results)
			}
			live = nil
			require.NoError(t, app.DB().Where("bucket_id = ? AND name = ?", bucket.ID, "same.txt").Find(&live).Error)
			assert.Len(t, live, 1)
		})
	}
}

func TestShareCancellationRefundsUploadLimit(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "share-quota@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "quota")
			limit := 1
			share := app.CreateShare(t, token, bucket.ID.String(), models.ShareCreateBody{
				Name:        "quota",
				Type:        models.ShareTypeBucket,
				Password:    sharePassword33,
				AllowUpload: true,
				MaxUploads:  &limit,
			})
			_, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
			base := fmt.Sprintf("/api/v1/shares/%s/files", share.ID)
			var transfer models.FileUploadResponse
			body := models.ShareUploadBody{Name: "quota.txt", Size: 4}
			require.Equal(t, http.StatusCreated, app.DoPublicShare(t, http.MethodPost, base, cookie, body, &transfer))
			cancelPath := base + "/" + transfer.ID + "/versions/" + transfer.VersionID
			for range 2 {
				require.Equal(
					t,
					http.StatusNoContent,
					app.DoPublicShare(t, http.MethodDelete, cancelPath, cookie, nil, nil),
				)
			}
			require.Equal(t, http.StatusCreated, app.DoPublicShare(t, http.MethodPost, base, cookie, body, &transfer))
			app.PutPresigned(t, transfer, []byte("next"))
			require.Equal(
				t,
				http.StatusNoContent,
				app.DoPublicShare(t, http.MethodPatch, base+"/"+transfer.ID, cookie, nil, nil),
			)
		})
	}
}
