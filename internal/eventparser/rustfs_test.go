package eventparser

import (
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
)

func rustfsMessage(eventName, key string) *message.Message {
	payload := fmt.Sprintf(
		`{"Records":[{"event_name":%q,"data":{"s3":{"bucket":{"name":"safebucket"},"object":{"key":%q}}}}]}`,
		eventName, key,
	)
	return message.NewMessage("id", []byte(payload))
}

func TestRustFSEventParser_GetBucketEventType(t *testing.T) {
	parser := &RustFSEventParser{}

	t.Run("CompleteMultipartUpload is classified as upload", func(t *testing.T) {
		msg := rustfsMessage("s3:ObjectCreated:CompleteMultipartUpload", "buckets/bucket-1/file-1")
		require.Equal(t, BucketEventTypeUpload, parser.GetBucketEventType(msg))
	})

	t.Run("Put is still classified as upload", func(t *testing.T) {
		msg := rustfsMessage("s3:ObjectCreated:Put", "buckets/bucket-1/file-1")
		require.Equal(t, BucketEventTypeUpload, parser.GetBucketEventType(msg))
	})

	t.Run("trash-prefixed CompleteMultipartUpload is ignored", func(t *testing.T) {
		msg := rustfsMessage("s3:ObjectCreated:CompleteMultipartUpload", "trash/bucket-1/files/file-1")
		require.Equal(t, BucketEventTypeIgnore, parser.GetBucketEventType(msg))
	})
}
