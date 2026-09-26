//go:build integration

package bucket_test

import (
	"fmt"
	"io"
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

func TestFileVersionUploads(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.LoadScenario(t, scenario)
			cfg.App.MaxFileVersions = 2
			app := bootstrap.BootTestApp(t, cfg)
			owner := app.CreateUser(t, "versions@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "versions")
			base := fmt.Sprintf("/api/v1/buckets/%s/files", bucket.ID)
			expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
			var first models.FileUploadResponse
			require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost, base, token,
				models.FileUploadBody{Name: "report.txt", Size: 3, ExpiresAt: &expires}, &first))
			assert.Equal(t, first.ID, first.VersionID)
			app.PutPresigned(t, first, []byte("one"))
			confirm := func(transfer models.FileUploadResponse) {
				t.Helper()
				require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPatch,
					base+"/"+transfer.ID+"/versions/"+transfer.VersionID, token, nil))
			}
			confirm(first)
			confirm(first)
			var second models.FileUploadResponse
			require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost, base, token,
				models.FileUploadBody{Name: "report.txt", Size: 3}, &second))
			assert.Equal(t, first.ID, second.ID)
			assert.NotEqual(t, first.VersionID, second.VersionID)
			require.Equal(t, http.StatusConflict, app.DoStatus(t, http.MethodPost, base, token,
				models.FileUploadBody{Name: "report.txt", Size: 3}))
			var download models.FileDownloadResponse
			require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, base+"/"+first.ID+"/url", token, nil, &download))
			assertDownload(t, download.URL, "one")
			app.PutPresigned(t, second, []byte("two"))
			require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPatch,
				base+"/"+second.ID, token, models.FilePatchBody{Status: "uploaded"}))
			var versions []models.FileVersionResponse
			require.Equal(
				t,
				http.StatusOK,
				app.Do(t, http.MethodGet, base+"/"+first.ID+"/versions", token, nil, &versions),
			)
			require.Len(t, versions, 2)
			assert.True(t, versions[0].IsCurrent)
			assert.Equal(t, 2, versions[0].Version)
			assert.Equal(t, &owner.ID, versions[0].UploadedBy)
			require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPut, base+"/"+first.ID, token,
				models.FileVersionRestoreBody{VersionID: uuid.MustParse(first.VersionID)}))
			require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, base+"/"+first.ID+"/url", token, nil, &download))
			assertDownload(t, download.URL, "one")

			share := app.CreateShare(t, token, bucket.ID.String(), models.ShareCreateBody{
				Name: "versions", Type: models.ShareTypeBucket, Password: sharePassword33, AllowUpload: true,
			})
			_, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
			require.NotEmpty(t, cookie)
			shareBase := fmt.Sprintf("/api/v1/shares/%s/files", share.ID)
			var third models.FileUploadResponse
			require.Equal(t, http.StatusCreated, app.DoPublicShare(t, http.MethodPost, shareBase, cookie,
				models.ShareUploadBody{Name: "report.txt", Size: 5}, &third))
			assert.Equal(t, first.ID, third.ID)
			app.PutPresigned(t, third, []byte("three"))
			shareVersion := shareBase + "/" + third.ID + "/versions/" + third.VersionID
			assertState := func() {
				t.Helper()
				var file models.File
				require.NoError(t, app.DB().Unscoped().Where("id = ?", first.ID).Find(&file).Error)
				assert.Equal(t, models.FileStatusUploaded, file.Status)
				require.NotNil(t, file.ExpiresAt)
				assert.True(t, expires.Equal(*file.ExpiresAt))
			}
			assertState()
			require.Equal(
				t,
				http.StatusNoContent,
				app.DoPublicShare(t, http.MethodPatch, shareBase+"/"+third.ID, cookie, nil, nil),
			)
			require.Equal(
				t,
				http.StatusNoContent,
				app.DoPublicShare(t, http.MethodPatch, shareVersion, cookie, nil, nil),
			)
			assertState()
			require.Equal(
				t,
				http.StatusOK,
				app.DoPublicShare(t, http.MethodGet, shareBase+"/"+first.ID+"/url", cookie, nil, &download),
			)
			assertDownload(t, download.URL, "three")
			versions = nil
			require.Equal(
				t,
				http.StatusOK,
				app.Do(t, http.MethodGet, base+"/"+first.ID+"/versions", token, nil, &versions),
			)
			require.Len(t, versions, 2)
			assert.Equal(t, 3, versions[0].Version)
			assert.Nil(t, versions[0].UploadedBy)
			assert.Equal(t, 2, versions[1].Version)
			manager := fileversions.Manager{DB: app.DB(), Storage: app.Storage, Cache: app.Cache}
			require.NoError(
				t,
				manager.Cleanup(bucket.ID, uuid.MustParse(first.ID), uuid.MustParse(first.VersionID)),
			)
			require.Equal(t, http.StatusNoContent,
				app.DoPublicShare(t, http.MethodPatch, shareBase+"/"+third.ID, cookie, nil, nil))
			require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPatch,
				base+"/"+third.ID, token, models.FilePatchBody{Status: "uploaded"}))
			var cancelled models.FileUploadResponse
			require.Equal(t, http.StatusCreated, app.DoPublicShare(t, http.MethodPost, shareBase, cookie,
				models.ShareUploadBody{Name: "report.txt", Size: 4}, &cancelled))
			cancelPath := shareBase + "/" + cancelled.ID + "/versions/" + cancelled.VersionID
			require.Equal(
				t,
				http.StatusNoContent,
				app.DoPublicShare(t, http.MethodDelete, cancelPath, cookie, nil, nil),
			)
			require.Equal(t, http.StatusConflict, app.DoPublicShare(t, http.MethodPatch, cancelPath, cookie, nil, nil))
			require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, base+"/"+first.ID+"/url", token, nil, &download))
			assertDownload(t, download.URL, "three")
		})
	}
}

func TestConcurrentVersionInitialization(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)
			owner := app.CreateUser(t, "concurrent-versions@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "concurrent")
			base := fmt.Sprintf("/api/v1/buckets/%s/files", bucket.ID)
			start := make(chan struct{})
			results := make(chan int, 2)
			for range 2 {
				go func() {
					<-start
					results <- app.DoStatus(t, http.MethodPost, base, token, models.FileUploadBody{Name: "same.txt", Size: 5})
				}()
			}
			close(start)
			assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, []int{<-results, <-results})
		})
	}
}

func assertDownload(t *testing.T, url, expected string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}
