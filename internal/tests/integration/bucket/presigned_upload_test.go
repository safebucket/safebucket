//go:build integration

package bucket_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mib             = 1024 * 1024
	multipartSize   = 65 * mib // two parts: 64 MiB + 1 MiB
	largeUploadCap  = int64(200 * mib)
	sharePassword33 = "horsebatterystaple"
)

func putPart(t *testing.T, url string, content []byte) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, url, bytes.NewReader(content))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	return resp.StatusCode
}

func multipartContent() []byte {
	return bytes.Repeat([]byte("x"), multipartSize)
}

func TestPresignedUpload(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.LoadScenario(t, scenario)
			cfg.App.MaxUploadSize = largeUploadCap
			app := bootstrap.BootTestApp(t, cfg)

			owner := app.CreateUser(t, "presign-owner@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "presign-bucket")
			bucketID := bucket.ID.String()

			content := multipartContent()
			part1 := content[:64*mib]
			part2 := content[64*mib:]

			t.Run("single PUT below the multipart threshold", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "small.txt", Size: 5}, &transfer)
				require.Equal(t, http.StatusCreated, status)
				assert.Equal(t, "put", transfer.Method)
				require.Len(t, transfer.Parts, 1)
				assert.NotEmpty(t, transfer.Parts[0].URL)
				assert.NotEmpty(t, transfer.Parts[0].Headers)
				assert.Equal(t, int64(5), transfer.Parts[0].Size)
				assert.Empty(t, transfer.Body)
				assert.Empty(t, transfer.URL)

				app.PutPresigned(t, transfer, []byte("test!"))

				require.Equal(t, http.StatusNoContent,
					app.DoStatus(t, http.MethodPatch,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID), token,
						models.FilePatchBody{Status: string(models.FileStatusUploaded)}))

				var dl models.FileDownloadResponse
				require.Equal(t, http.StatusOK,
					app.Do(t, http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s/url", bucketID, transfer.ID), token, nil, &dl))
				assert.NotEmpty(t, dl.URL)
			})

			t.Run("multipart above the threshold", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "big.bin", Size: multipartSize}, &transfer)
				require.Equal(t, http.StatusCreated, status)
				require.Equal(t, "put", transfer.Method)
				require.Len(t, transfer.Parts, 2)
				assert.Equal(t, int64(64*mib), transfer.Parts[0].Size)
				assert.Equal(t, int64(mib), transfer.Parts[1].Size)

				require.Less(t, putPart(t, transfer.Parts[0].URL, part1), 300)
				require.Less(t, putPart(t, transfer.Parts[1].URL, part2), 300)

				require.Equal(t, http.StatusNoContent,
					app.DoStatus(t, http.MethodPatch,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID), token,
						models.FilePatchBody{Status: string(models.FileStatusUploaded)}))

				var dl models.FileDownloadResponse
				require.Equal(t, http.StatusOK,
					app.Do(t, http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s/url", bucketID, transfer.ID), token, nil, &dl))
				assert.NotEmpty(t, dl.URL)
			})

			t.Run("multipart with a missing part is rejected", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "missing-part.bin", Size: multipartSize}, &transfer)
				require.Equal(t, http.StatusCreated, status)
				require.Len(t, transfer.Parts, 2)

				require.Less(t, putPart(t, transfer.Parts[0].URL, part1), 300)

				code, errs := app.DoExpectError(t, http.MethodPatch,
					fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID), token,
					models.FilePatchBody{Status: string(models.FileStatusUploaded)})
				assert.Equal(t, http.StatusBadRequest, code)
				assert.Contains(t, errs, apierrors.CodeMultipartSizeMismatch)
			})

			t.Run("double confirm returns conflict", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "double.bin", Size: multipartSize}, &transfer)
				require.Equal(t, http.StatusCreated, status)

				require.Less(t, putPart(t, transfer.Parts[0].URL, part1), 300)
				require.Less(t, putPart(t, transfer.Parts[1].URL, part2), 300)

				patchPath := fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID)
				require.Equal(t, http.StatusNoContent,
					app.DoStatus(t, http.MethodPatch, patchPath, token,
						models.FilePatchBody{Status: string(models.FileStatusUploaded)}))
				require.Equal(t, http.StatusConflict,
					app.DoStatus(t, http.MethodPatch, patchPath, token,
						models.FilePatchBody{Status: string(models.FileStatusUploaded)}))
			})

			t.Run("cancel an in-flight multipart upload", func(t *testing.T) {
				var transfer models.FileUploadResponse
				status := app.Do(t, http.MethodPost, fmt.Sprintf("/api/v1/buckets/%s/files", bucketID), token,
					models.FileUploadBody{Name: "cancel.bin", Size: multipartSize}, &transfer)
				require.Equal(t, http.StatusCreated, status)

				require.Less(t, putPart(t, transfer.Parts[0].URL, part1), 300)

				require.Equal(t, http.StatusNoContent,
					app.DoStatus(t, http.MethodDelete,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketID, transfer.ID), token, nil))

				require.Equal(t, http.StatusNotFound,
					app.DoStatus(t, http.MethodGet,
						fmt.Sprintf("/api/v1/buckets/%s/files/%s/url", bucketID, transfer.ID), token, nil))
			})
		})
	}
}

func TestPresignedShareUpload(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.LoadScenario(t, scenario)
			cfg.App.MaxUploadSize = largeUploadCap
			app := bootstrap.BootTestApp(t, cfg)

			owner := app.CreateUser(t, "presign-share-owner@example.com")
			token := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, token, "presign-share-bucket")

			content := multipartContent()
			part1 := content[:64*mib]
			part2 := content[64*mib:]

			t.Run("multipart upload through a share", func(t *testing.T) {
				share := app.CreateShare(t, token, bucket.ID.String(), models.ShareCreateBody{
					Name:        "upload-share",
					Type:        models.ShareTypeBucket,
					Password:    sharePassword33,
					AllowUpload: true,
				})
				authStatus, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
				require.Equal(t, http.StatusOK, authStatus)
				require.NotEmpty(t, cookie)

				var transfer models.FileUploadResponse
				status := app.DoPublicShare(t, http.MethodPost,
					fmt.Sprintf("/api/v1/shares/%s/files", share.ID), cookie,
					models.ShareUploadBody{Name: "share-big.bin", Size: multipartSize}, &transfer)
				require.Equal(t, http.StatusCreated, status)
				require.Equal(t, "put", transfer.Method)
				require.Len(t, transfer.Parts, 2)

				require.Less(t, putPart(t, transfer.Parts[0].URL, part1), 300)
				require.Less(t, putPart(t, transfer.Parts[1].URL, part2), 300)

				require.Equal(t, http.StatusNoContent,
					app.DoPublicShare(t, http.MethodPatch,
						fmt.Sprintf("/api/v1/shares/%s/files/%s", share.ID, transfer.ID), cookie, nil, nil))
			})

			t.Run("upload is rejected when the share disallows it", func(t *testing.T) {
				share := app.CreateShare(t, token, bucket.ID.String(), models.ShareCreateBody{
					Name:        "no-upload-share",
					Type:        models.ShareTypeBucket,
					Password:    sharePassword33,
					AllowUpload: false,
				})
				_, cookie := app.AuthenticateShare(t, share.ID.String(), sharePassword33)
				require.NotEmpty(t, cookie)

				status := app.DoPublicShare(t, http.MethodPost,
					fmt.Sprintf("/api/v1/shares/%s/files", share.ID), cookie,
					models.ShareUploadBody{Name: "nope.txt", Size: 5}, nil)
				assert.Equal(t, http.StatusForbidden, status)
			})
		})
	}
}
