package models

type Activity struct {
	Message string
	Filter  LogFilter
}

type LogFilter struct {
	Fields    map[string]string
	Timestamp string
}

type LokiQueryResponse struct {
	Data struct {
		Result []LokiResult `json:"result"`
	} `json:"data"`
}

type LokiResult struct {
	Stream map[string]string `json:"stream"` // dynamic label key-value pairs
	Values [][2]string       `json:"values"` // each value is a [timestamp, logLine]
}

type History struct {
	Action     string `json:"action"`
	BucketId   string `json:"bucket_id"`
	Domain     string `json:"domain"`
	ObjectType string `json:"object_type"`
	UserId     string `json:"user_id"`
	Timestamp  string `json:"timestamp"`
	Message    string `json:"message"`
}
