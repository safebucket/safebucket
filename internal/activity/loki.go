package activity

import (
	"api/internal/models"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var authorizedLabels = [3]string{"domain", "object_type", "action"}

const lokiPushURI = "/loki/api/v1/push"
const lokiSearchURI = "/loki/api/v1/query_range"

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

type LokiQueryResponse struct {
	Data struct {
		Result []LokiResult `json:"result"`
	} `json:"data"`
}

type LokiResult struct {
	Stream map[string]string `json:"stream"` // dynamic label key-value pairs
	Values [][2]string       `json:"values"` // each value is a [timestamp, logLine]
}

// RawLogValue is a fixed-size array of 3 interface{} elements, typically representing [timestamp, message, metadata].
type RawLogValue [3]interface{}

// LokiClient provides methods to interact with a Loki logging endpoint, including sending logs and searching for logs.
type LokiClient struct {
	Client    *resty.Client
	pushURL   string
	searchURL string
}

func (s *LokiClient) Send(activity models.Activity) error {
	lokiBody := createLokiBody(activity)

	resp, err := s.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(lokiBody).
		Post(s.pushURL)

	if err != nil {
		zap.L().Error("Failed to send data to loki ", zap.Any("error", err))
		return err
	}

	if resp.StatusCode() != 204 {
		zap.L().Error("Failed to send data to loki ", zap.Int("status_code", resp.StatusCode()))
		return err
	}

	return nil
}

func (s *LokiClient) Search(searchCriteria map[string][]string) ([]map[string]interface{}, error) {
	query := generateSearchQuery(searchCriteria)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()
	resp, err := s.Client.R().
		SetQueryParams(map[string]string{"start": strconv.FormatInt(thirtyDaysAgo, 10), "limit": "100", "query": query}).
		SetHeader("Accept", "application/json").
		Get(s.searchURL)

	if err != nil {
		zap.L().Error("Failed to query Loki", zap.Any("error", err))
		return []map[string]interface{}{}, err
	}

	if resp.StatusCode() != 200 {
		zap.L().Error("Failed to query data from loki ", zap.Int("status_code", resp.StatusCode()))
		return []map[string]interface{}{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	var parsedResp LokiQueryResponse
	if err := json.Unmarshal(resp.Body(), &parsedResp); err != nil {
		zap.L().Error("Failed to parse Loki response", zap.Error(err))
		return []map[string]interface{}{}, err
	}

	var activity []map[string]interface{}
	for _, log := range parsedResp.Data.Result {
		var entry = map[string]interface{}{
			"domain":      log.Stream["domain"],
			"user_id":     log.Stream["user_id"],
			"action":      log.Stream["action"],
			"object_type": log.Stream["object_type"],
			"bucket_id":   log.Stream["bucket_id"],
			"timestamp":   log.Values[0][0],
			"message":     log.Values[0][1],
		}

		activity = append(activity, entry)
	}

	return activity, nil
}

// NewLokiClient initializes and returns a new LokiClient instance based on the provided log configuration.
func NewLokiClient(config models.ActivityConfiguration) IActivityLogger {
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

	return &LokiClient{
		Client:    client,
		pushURL:   fmt.Sprintf("%s%s", config.Endpoint, lokiPushURI),
		searchURL: fmt.Sprintf("%s%s", config.Endpoint, lokiSearchURI),
	}
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
func splitMetadata[T interface{}](structuredMetadata map[string]T) (map[string]T, map[string]T) {
	labels := make(map[string]T)
	metadata := make(map[string]T)

	for key, value := range structuredMetadata {
		if isAuthorized(key) {
			labels[key] = value
		} else {
			metadata[key] = value
		}
	}
	return labels, metadata
}

func generateSearchQuery(searchCriteria map[string][]string) string {
	labels, metadata := splitMetadata(searchCriteria)

	formattedLabels := generateORCriteria(labels)
	formattedMetadata := generateORCriteria(metadata)

	return fmt.Sprintf(
		"{%s} | %s",
		strings.Join(formattedLabels, ", "),
		strings.Join(formattedMetadata, " | "),
	)
}

func generateORCriteria(criteria map[string][]string) []string {
	var formattedCriteria []string
	for key, value := range criteria {
		joinedValue := strings.Join(value, "|")
		formattedCriteria = append(formattedCriteria, fmt.Sprintf("%s=~\"%s\"", key, joinedValue))
	}

	return formattedCriteria
}

// createLokiBody transforms a LogMessage into a LokiBody structure, separating metadata into labels and additional fields.
// It constructs a Loki-compatible log entry stream with the message and associated metadata.
// Returns the generated LokiBody and any error encountered during its creation.
func createLokiBody(activity models.Activity) LokiBody {
	labels, metadata := splitMetadata(activity.Filter.Fields)
	entry := RawLogValue{activity.Filter.Timestamp, activity.Message, metadata}
	stream := StreamEntry{
		Stream: labels,
		Values: []RawLogValue{entry},
	}

	return LokiBody{
		Streams: []StreamEntry{stream},
	}
}
