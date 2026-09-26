package storage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrashUsesCurrentVersionContentAndFileMarker(t *testing.T) {
	for _, provider := range []string{"rustfs", "aws"} {
		t.Run(provider, func(t *testing.T) {
			versionID := uuid.New()
			file := models.File{ID: uuid.New(), BucketID: uuid.New(), CurrentVersionID: &versionID}
			objectPath := "buckets/" + file.BucketID.String() + "/" + file.ID.String()
			contentPath := "/test-bucket/buckets/" + file.BucketID.String() + "/" + versionID.String()
			markerPath := "/test-bucket/trash/" + file.BucketID.String() + "/files/" + file.ID.String()
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				if r.Method == http.MethodHead && r.URL.Path == contentPath {
					w.Header().Set("Content-Length", "4")
					w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
					w.Header().Set("ETag", "abcd")
					return
				}
				if r.Method == http.MethodPut && r.URL.Path == markerPath {
					w.Header().Set("ETag", "abcd")
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()
			var store IStorage
			if provider == "rustfs" {
				client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
					Creds: credentials.NewStaticV4("test", "test", ""), Region: "us-east-1",
				})
				require.NoError(t, err)
				store = RustFSStorage{BucketName: "test-bucket", storage: client}
			} else {
				store = AWSStorage{BucketName: "test-bucket", storage: testAWSClient(server.URL)}
			}
			require.NoError(t, store.MarkAsTrashed(objectPath, file))
			assert.Equal(t, []string{"HEAD " + contentPath, "PUT " + markerPath}, requests)
		})
	}
}

func TestAWSBatchDeleteReportsObjectFailures(t *testing.T) {
	failed := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/xml")
		if failed {
			_, _ = w.Write(
				[]byte(
					`<DeleteResult><Error><Key>blocked</Key><Code>AccessDenied</Code><Message>denied</Message></Error></DeleteResult>`,
				),
			)
			return
		}
		_, _ = w.Write([]byte(`<DeleteResult/>`))
	}))
	defer server.Close()
	store := AWSStorage{BucketName: "test-bucket", storage: testAWSClient(server.URL)}
	err := store.RemoveObjects([]string{"removed", "blocked"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "blocked")
	assert.ErrorContains(t, err, "AccessDenied")
	failed = false
	require.NoError(t, store.RemoveObjects([]string{"removed", "blocked"}))
}

func testAWSClient(endpoint string) *s3.Client {
	return s3.New(s3.Options{
		Region: "us-east-1", BaseEndpoint: aws.String(endpoint), UsePathStyle: true,
		Credentials: aws.AnonymousCredentials{},
	})
}
