package eventparser

import (
	"encoding/json"
	"testing"

	"github.com/safebucket/safebucket/internal/storage"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

func TestUploadVersionMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		parser  IBucketEventParser
		payload string
	}{
		{"minio", &MinIOEventParser{}, `{"Records":[{"s3":{"object":{"userMetadata":{"X-Amz-Meta-Bucket-Id":"bucket","X-Amz-Meta-File-Id":"file","X-Amz-Meta-Version-Id":"version","X-Amz-Meta-User-Id":"user"}}}}]}`},
		{"rustfs", &RustFSEventParser{}, `{"Records":[{"data":{"s3":{"object":{"userMetadata":{"bucket-id":"bucket","file-id":"file","version-id":"version","user-id":"user"}}}}}]}`},
		{"aws", &AWSEventParser{Storage: versionMetadataStorage{}}, `{"Records":[{"s3":{"object":{"key":"buckets/bucket/version"}}}]}`},
		{"azure", &AzureEventParser{Storage: versionMetadataStorage{}}, `[{"eventType":"Microsoft.Storage.BlobCreated","subject":"/blobServices/default/containers/test/blobs/buckets/bucket/version"}]`},
		{"gcp", &GCPEventParser{}, `{"metadata":{"bucket-id":"bucket","file-id":"file","version-id":"version","user-id":"user"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.True(t, json.Valid([]byte(test.payload)))
			msg := message.NewMessage("event", []byte(test.payload))
			msg.Metadata.Set("eventType", "OBJECT_FINALIZE")
			events := test.parser.ParseBucketUploadEvents(msg)
			require.Len(t, events, 1)
			require.Equal(t, "version", events[0].VersionID)
			require.Equal(t, "file", events[0].FileID)
		})
	}
}

type versionMetadataStorage struct{ storage.IStorage }

func (versionMetadataStorage) StatObject(string) (map[string]string, error) {
	return map[string]string{"bucket_id": "bucket", "file_id": "file", "version_id": "version", "user_id": "user"}, nil
}
