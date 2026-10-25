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

func TestFileVersionUploader(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.WithLocalSharing(bootstrap.LoadScenario(t, scenario), true)
			app := bootstrap.BootTestApp(t, cfg)

			owner := app.CreateUser(t, "version-uploader-owner@example.com")
			contributor := app.CreateUser(t, "version-uploader-contributor@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			contributorToken := app.LoginAs(t, contributor.Email)

			bucket := app.CreateBucket(t, ownerToken, "version-uploader-bucket")
			bucketID := bucket.ID.String()
			app.AddMembers(t, ownerToken, bucketID, []models.BucketMemberBody{
				{Email: contributor.Email, Group: models.GroupContributor},
			})

			fileID := app.UploadTestFile(t, ownerToken, bucketID, "shared.txt")

			var transfer models.FileUploadResponse
			status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID),
				contributorToken, models.FileUploadBody{Name: "shared.txt", Size: 7}, &transfer)
			require.Equal(t, http.StatusCreated, status)
			app.PutPresigned(t, transfer, []byte("seven!!"))

			require.Equal(t, http.StatusNoContent,
				app.DoStatus(t, http.MethodPatch,
					fmt.Sprintf("/api/v1/buckets/%s/files/%s/versions/%s", bucketID, transfer.ID, transfer.VersionID),
					contributorToken, nil))

			var versions []models.FileVersionResponse
			require.Equal(t, http.StatusOK,
				app.Do(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s/files/%s/versions", bucketID, fileID), ownerToken, nil, &versions))

			latest := versionByNumber(t, versions, 2)
			require.NotNil(t, latest.UploadedBy)
			assert.Equal(t, contributor.ID, latest.UploadedBy.ID)
			assert.Equal(t, contributor.Email, latest.UploadedBy.Email)
			assert.Equal(t, "Test", latest.UploadedBy.FirstName)
			assert.Equal(t, "User", latest.UploadedBy.LastName)

			first := versionByNumber(t, versions, 1)
			require.NotNil(t, first.UploadedBy)
			assert.Equal(t, owner.ID, first.UploadedBy.ID)
			assert.Equal(t, owner.Email, first.UploadedBy.Email)

			t.Run("anonymous share upload has no uploader", func(t *testing.T) {
				share := app.CreateShare(t, ownerToken, bucketID, models.ShareCreateBody{
					Name:        "version-uploader-share",
					Type:        models.ShareTypeBucket,
					Password:    sharePassword33,
					AllowUpload: true,
				})

				authStatus, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
				require.Equal(t, http.StatusOK, authStatus)
				require.NotEmpty(t, cookie)

				var shareTransfer models.FileUploadResponse
				require.Equal(t, http.StatusCreated,
					app.DoPublicShare(t, http.MethodPost,
						fmt.Sprintf("/api/v1/shares/%s/files", share.ID), cookie,
						models.ShareUploadBody{Name: "anonymous.txt", Size: 5}, &shareTransfer))
				app.PutPresigned(t, shareTransfer, []byte("test!"))

				require.Equal(t, http.StatusNoContent,
					app.DoPublicShare(t, http.MethodPatch,
						fmt.Sprintf("/api/v1/shares/%s/files/%s", share.ID, shareTransfer.ID), cookie, nil, nil))

				var anonymousVersions []models.FileVersionResponse
				require.Equal(t, http.StatusOK,
					app.Do(t, http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s/versions", bucketID, shareTransfer.ID),
						ownerToken, nil, &anonymousVersions))
				require.Len(t, anonymousVersions, 1)
				assert.Nil(t, anonymousVersions[0].UploadedBy)
			})
		})
	}
}

func versionByNumber(t *testing.T, versions []models.FileVersionResponse, number int) models.FileVersionResponse {
	t.Helper()

	for _, version := range versions {
		if version.Version == number {
			return version
		}
	}
	t.Fatalf("version %d not found", number)
	return models.FileVersionResponse{}
}
