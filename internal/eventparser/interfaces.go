package eventparser

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	BucketEventTypeUpload  = "upload"
	BucketEventTypeUnknown = "unknown"
	BucketEventTypeIgnore  = "ignore"
)

type IBucketEventParser interface {
	GetBucketEventType(*message.Message) string
	ParseBucketUploadEvents(*message.Message) []BucketUploadEvent
}
