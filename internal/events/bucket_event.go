package events

type S3Event struct {
	EventName string   `json:"EventName"`
	Key       string   `json:"Key"`
	Records   []Record `json:"Records"`
}

type Record struct {
	EventName string `json:"eventName"`
	EventTime string `json:"eventTime"`
	S3        S3Info `json:"s3"`
}

type S3Info struct {
	Bucket BucketInfo `json:"bucket"`
	Object ObjectInfo `json:"object"`
}

type BucketInfo struct {
	Name string `json:"name"`
}

type ObjectInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"contentType"`
	UserMetadata map[string]string `json:"userMetadata"`
}
