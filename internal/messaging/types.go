package messaging

type BucketUploadEvent struct {
	BucketId string `json:"bucket_id"`
	FileId   string `json:"file_id"`
	UserId   string `json:"user_id"`
}

type BucketDeletionEvent struct {
	BucketId  string `json:"bucket_id"`
	ObjectKey string `json:"object_key"`
	EventName string `json:"event_name"`
}

type MinioEvent struct {
	Records []struct {
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
	} `json:"Records"`
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
