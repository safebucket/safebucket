package messaging

type BucketUploadEvent struct {
	BucketID string `json:"bucket_id"`
	FileID   string `json:"file_id"`
	UserID   string `json:"user_id"`
}

type BucketDeletionEvent struct {
	BucketID  string `json:"bucket_id"`
	ObjectKey string `json:"object_key"`
	EventName string `json:"event_name"`
}

type RustFSEvent struct {
	EventName string              `json:"EventName"`
	Key       string              `json:"Key"`
	Records   []RustFSEventRecord `json:"Records"`
}

type RustFSEventRecord struct {
	ObjectName string          `json:"object_name"`
	BucketName string          `json:"bucket_name"`
	EventName  string          `json:"event_name"`
	Data       RustFSEventData `json:"data"`
}

type RustFSEventData struct {
	EventName string `json:"eventName"`
	S3        struct {
		Bucket struct {
			Name string `json:"name"`
		} `json:"bucket"`
		Object struct {
			Key          string            `json:"key"`
			Size         int64             `json:"size"`
			ContentType  string            `json:"contentType"`
			UserMetadata map[string]string `json:"userMetadata"`
		} `json:"object"`
	} `json:"s3"`
}

type GCPEvent struct {
	Metadata map[string]string `json:"metadata"`
}

type AWSEvent struct {
	Records []struct {
		EventName string `json:"eventName"`
		S3        struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key  string `json:"key"`
				Size int64  `json:"size"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}
