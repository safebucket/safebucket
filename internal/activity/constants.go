package activity

import "api/internal/models"

const (
	BucketCreated  string = "BUCKET_CREATED"
	FileUploaded   string = "FILE_UPLOADED"
	FileDownloaded string = "FILE_DOWNLOADED"
	FileUpdated    string = "FILE_UPDATED"
	FileDeleted    string = "FILE_DELETED"
)

type ToEnrichValue struct {
	Name   string
	Object interface{}
}

var ToEnrich = map[string]ToEnrichValue{
	"user_id":   {Name: "user", Object: models.User{}},
	"bucket_id": {Name: "bucket", Object: models.Bucket{}},
}
