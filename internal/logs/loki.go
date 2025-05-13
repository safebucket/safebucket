package logs

import (
	"api/internal/models"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"time"
)

var authorizedLabels = [3]string{"domain", "obj_type", "action"}

const lokiPushURI = "/loki/api/v1/push"
const lokiSearchURI = "/loki/api/v1/query"

// LokiBody represents the main structure for sending logs to Loki, containing a list of log stream entries.
type LokiBody struct {
	Streams []StreamEntry `json:"streams"`
}

// StreamEntry represents a single log record stream in a Loki-compatible format.
// The Stream field contains dynamic labels as key-value pairs.
// The Values field contains log entries with their associated timestamp and message.
type StreamEntry struct {
	Stream map[string]string `json:"stream"` // dynamic labels like "foo": "bar2"
	Values []RawLogValue     `json:"values"` // each entry is a [timestamp, message]
}

// RawLogValue is a fixed-size array of 3 interface{} elements, typically representing [timestamp, message, metadata].
type RawLogValue [3]interface{}

// LokiClient provides methods to interact with a Loki logging endpoint, including sending logs and searching for logs.
type LokiClient struct {
	Endpoint  string
	pushURL   string
	searchURL string
}

func (s *LokiClient) Send(log models.LogMessage) error {

	lokiBody, err := createLokiBody(log)
	if err != nil {
		return err
	}

	client := resty.New()
	client.SetRetryCount(5).
		SetRetryWaitTime(3 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry on network errors
			if err != nil {
				zap.L().Debug("Retrying due to network error", zap.Error(err))
				return true
			}

			// Retry on server errors (5xx)
			if r.StatusCode() >= 500 {
				zap.L().Debug("Retrying due to server error",
					zap.Int("statusCode", r.StatusCode()),
					zap.String("status", r.Status()))
				return true
			}

			// Retry on specific error codes that might be temporary
			// 429: Too Many Requests
			// 408: Request Timeout
			if r.StatusCode() == 429 || r.StatusCode() == 408 {
				zap.L().Debug("Retrying due to rate limiting or timeout",
					zap.Int("statusCode", r.StatusCode()),
					zap.String("status", r.Status()))
				return true
			}
			return false
		})

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(lokiBody).
		Post(s.pushURL)

	if err != nil {
		zap.L().Error("Failed to send data to loki ", zap.Any("error", err))
		return err
	}

	if resp.StatusCode() != 204 {
		zap.L().Error("Failed to send data to loki ", zap.Any("status_code", resp.StatusCode()))
		return err
	}

	return nil
}

func (s *LokiClient) Search(key map[string]string) error {
	panic("implement me")
}

// isAuthorized checks if the given label is part of the predefined authorizedLabels array and returns true if matched.
func isAuthorized(label string) bool {
	for _, auth := range authorizedLabels {
		if label == auth {
			return true
		}
	}
	return false
}

// splitMetadata separates a map into labels and metadata based on specific authorization criteria.
// Returns two maps: labels containing authorized keys and metadata containing unauthorized keys.
func splitMetadata(structuredMetadata map[string]string) (map[string]string, map[string]string) {
	labels := make(map[string]string)
	metadata := make(map[string]string)

	for key, value := range structuredMetadata {
		if isAuthorized(key) {
			labels[key] = value
		} else {
			metadata[key] = value
		}
	}
	return labels, metadata
}

// createLokiBody transforms a LogMessage into a LokiBody structure, separating metadata into labels and additional fields.
// It constructs a Loki-compatible log entry stream with the message and associated metadata.
// Returns the generated LokiBody and any error encountered during its creation.
func createLokiBody(log models.LogMessage) (LokiBody, error) {
	labels, metadata := splitMetadata(log.Filter.Fields)
	entry := RawLogValue{log.Filter.Timestamp, log.Message, metadata}
	stream := StreamEntry{
		Stream: labels,
		Values: []RawLogValue{entry},
	}
	body := LokiBody{
		Streams: []StreamEntry{stream},
	}
	return body, nil
}

// NewLokiClient initializes and returns a new LokiClient instance based on the provided log configuration.
func NewLokiClient(config models.LogConfiguration) ILogClient {
	return &LokiClient{
		Endpoint:  config.Loki.Endpoint,
		pushURL:   fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiPushURI),
		searchURL: fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiSearchURI),
	}
}
