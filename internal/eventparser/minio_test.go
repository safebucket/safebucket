package eventparser

import (
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

func minioMessage(eventName, key string) *message.Message {
	payload := fmt.Sprintf(
		`{"Records":[{"eventName":%q,"s3":{"bucket":{"name":"safebucket"},"object":{"key":%q}}}]}`,
		eventName, key,
	)
	return message.NewMessage("id", []byte(payload))
}

func TestMinIOEventParser_GetBucketEventType(t *testing.T) {
	parser := &MinIOEventParser{}

	t.Run("CompleteMultipartUpload is classified as upload", func(t *testing.T) {
		msg := minioMessage("s3:ObjectCreated:CompleteMultipartUpload", "buckets/bucket-1/file-1")
		require.Equal(t, BucketEventTypeUpload, parser.GetBucketEventType(msg))
	})

	t.Run("Put is still classified as upload", func(t *testing.T) {
		msg := minioMessage("s3:ObjectCreated:Put", "buckets/bucket-1/file-1")
		require.Equal(t, BucketEventTypeUpload, parser.GetBucketEventType(msg))
	})

	t.Run("trash-prefixed CompleteMultipartUpload is ignored", func(t *testing.T) {
		msg := minioMessage("s3:ObjectCreated:CompleteMultipartUpload", "trash/bucket-1/files/file-1")
		require.Equal(t, BucketEventTypeIgnore, parser.GetBucketEventType(msg))
	})
}
