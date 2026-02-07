package eventparser

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	BucketEventTypeUpload   = "upload"
	BucketEventTypeDeletion = "deletion"
	BucketEventTypeUnknown  = "unknown"
	BucketEventTypeIgnore   = "ignore"
)

// IBucketEventParser handles interpretation of bucket event payloads.
type IBucketEventParser interface {
	GetBucketEventType(*message.Message) string
	ParseBucketUploadEvents(*message.Message) []BucketUploadEvent
	ParseBucketDeletionEvents(*message.Message, string) []BucketDeletionEvent
}
