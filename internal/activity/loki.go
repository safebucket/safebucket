package activity

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/safebucket/safebucket/internal/configuration"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const (
	lokiPushURI   = "/loki/api/v1/push"
	lokiSearchURI = "/loki/api/v1/query_range"
)

type LokiBody struct {
	Streams []StreamEntry `json:"streams"`
}

type StreamEntry struct {
	Stream map[string]string `json:"stream"`
	Values []RawLogValue     `json:"values"`
}

type LokiQueryResponse struct {
	Data struct {
		Result []LokiResult `json:"result"`
	} `json:"data"`
}

type LokiResult struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type RawLogValue [3]interface{}

type LokiMatrixResponse struct {
	Data struct {
		ResultType string             `json:"resultType"`
		Result     []LokiMatrixResult `json:"result"`
	} `json:"data"`
}

type LokiMatrixResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

type LokiClient struct {
	Client    *resty.Client
	pushURL   string
	searchURL string
}

func (s *LokiClient) Close() error {
	return nil
}

func (s *LokiClient) Send(activity models.Activity) error {
	lokiBody, err := createLokiBody(activity)
	if err != nil {
		return err
	}

	resp, err := s.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(lokiBody).
		Post(s.pushURL)
	if err != nil {
		zap.L().Error("Failed to send data to loki ", zap.Any("error", err))
		return err
	}

	if resp.StatusCode() != 204 {
		zap.L().Error("Failed to send data to loki ",
			zap.Int("status_code", resp.StatusCode()),
			zap.String("response_body", string(resp.Body())))
		return fmt.Errorf("loki returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return nil
}

func (s *LokiClient) Search(searchCriteria map[string][]string) ([]map[string]interface{}, error) {
	query := generateSearchQuery(searchCriteria)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()
	params := map[string]string{
		"start":     strconv.FormatInt(thirtyDaysAgo, 10),
		"limit":     "100",
		"query":     query,
		"direction": "backward",
	}

	resp, err := s.Client.R().
		SetQueryParams(params).
		SetHeader("Accept", "application/json").
		Get(s.searchURL)
	if err != nil {
		zap.L().Error("Failed to query Loki", zap.Any("error", err))
		return []map[string]interface{}{}, err
	}

	if resp.StatusCode() != 200 {
		zap.L().Error("Failed to query data from loki",
			zap.String("query", query),
			zap.Int("status_code", resp.StatusCode()),
			zap.String("response_body", string(resp.Body())),
			zap.Error(err),
		)
		return []map[string]interface{}{}, fmt.Errorf(
			"unexpected status code: %d",
			resp.StatusCode(),
		)
	}

	var parsedResp LokiQueryResponse
	if err = json.Unmarshal(resp.Body(), &parsedResp); err != nil {
		zap.L().Error("Failed to parse Loki response",
			zap.String("query", query),
			zap.Int("status_code", resp.StatusCode()),
			zap.String("response_body", string(resp.Body())),
			zap.Error(err),
		)
		return []map[string]interface{}{}, err
	}

	var activity []map[string]interface{}
	for _, result := range parsedResp.Data.Result {
		for _, log := range result.Values {
			entry := map[string]interface{}{
				"domain":              result.Stream["domain"],
				"user_id":             result.Stream["user_id"],
				"action":              result.Stream["action"],
				"object_type":         result.Stream["object_type"],
				"bucket_id":           result.Stream["bucket_id"],
				"file_id":             result.Stream["file_id"],
				"folder_id":           result.Stream["folder_id"],
				"share_id":            result.Stream["share_id"],
				"bucket_member_email": result.Stream["bucket_member_email"],
				"timestamp":           log[0],
			}

			var logLineData map[string]interface{}
			if err = json.Unmarshal([]byte(log[1]), &logLineData); err != nil {
				zap.L().Error("Failed to unmarshal log line data from Loki",
					zap.Error(err),
					zap.String("log_line_str", log[1]))
				return activity, err
			}

			if message, ok := logLineData["message"]; ok {
				entry["message"] = message
			}
			if object, ok := logLineData["object"]; ok {
				entry["object"] = object
			}

			activity = append(activity, entry)
		}
	}

	return activity, nil
}

func (s *LokiClient) CountByDay(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error) {
	baseQuery := generateSearchQuery(searchCriteria)
	countQuery := fmt.Sprintf("sum(count_over_time(%s[1d]))", baseQuery)

	startTime := time.Now().AddDate(0, 0, -days)
	endTime := time.Now()

	params := map[string]string{
		"query": countQuery,
		"start": strconv.FormatInt(startTime.Unix(), 10),
		"end":   strconv.FormatInt(endTime.Unix(), 10),
		"step":  "1d",
	}

	zap.L().Debug("Search query", zap.String("query", countQuery), zap.Any("params", params))

	resp, err := s.Client.R().
		SetQueryParams(params).
		SetHeader("Accept", "application/json").
		Get(s.searchURL)
	if err != nil {
		zap.L().Error("Failed to query Loki for count by day", zap.Any("error", err))
		return nil, err
	}

	if resp.StatusCode() != 200 {
		zap.L().Error("Failed to get count by day from Loki",
			zap.String("query", countQuery),
			zap.Int("status_code", resp.StatusCode()),
			zap.String("response_body", string(resp.Body())),
		)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	var parsedResp LokiMatrixResponse
	if err = json.Unmarshal(resp.Body(), &parsedResp); err != nil {
		zap.L().Error("Failed to parse Loki matrix response",
			zap.String("query", countQuery),
			zap.Int("status_code", resp.StatusCode()),
			zap.String("response_body", string(resp.Body())),
			zap.Error(err),
		)
		return nil, err
	}

	if len(parsedResp.Data.Result) == 0 {
		return []models.TimeSeriesPoint{}, nil
	}

	values := parsedResp.Data.Result[0].Values
	result := make([]models.TimeSeriesPoint, 0, len(values))
	for _, value := range values {
		if len(value) < 2 {
			continue
		}

		timestamp, ok := value[0].(float64)
		if !ok {
			continue
		}

		countStr, ok := value[1].(string)
		if !ok {
			continue
		}

		count, err2 := strconv.ParseInt(countStr, 10, 64)
		if err2 != nil {
			continue
		}

		// Loki returns the END of the bucket as timestamp, subtract 1 day to get the actual date
		dateStr := time.Unix(int64(timestamp), 0).UTC().AddDate(0, 0, -1).Format("2006-01-02")

		result = append(result, models.TimeSeriesPoint{
			Date:  dateStr,
			Count: count,
		})
	}

	slices.SortFunc(result, func(a, b models.TimeSeriesPoint) int {
		return strings.Compare(a.Date, b.Date)
	})

	return result, nil
}

func NewLokiClient(config models.ActivityConfiguration) IActivityLogger {
	client := resty.New()
	client.SetRetryCount(5).
		SetRetryWaitTime(3 * time.Second).
		SetRetryMaxWaitTime(20 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if err != nil {
				zap.L().Debug("Retrying due to network error", zap.Error(err))
				return true
			}

			if r.StatusCode() >= 500 {
				zap.L().Debug("Retrying due to server error",
					zap.Int("statusCode", r.StatusCode()),
					zap.String("status", r.Status()))
				return true
			}

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
		pushURL:   fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiPushURI),
		searchURL: fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiSearchURI),
	}
}

func generateSearchQuery(searchCriteria map[string][]string) string {
	labels, metadata := lokiLabels(searchCriteria)

	formattedLabels := generateORCriteria(labels)
	formattedMetadata := generateORCriteria(metadata)

	query := fmt.Sprintf("{%s}", strings.Join(formattedLabels, ", "))

	if len(formattedMetadata) > 0 {
		query = fmt.Sprintf("%s | %s", query, strings.Join(formattedMetadata, " | "))
	}

	return query
}

func generateORCriteria(criteria map[string][]string) []string {
	var formattedCriteria []string
	for key, value := range criteria {
		joinedValue := strings.Join(value, "|")
		formattedCriteria = append(formattedCriteria, fmt.Sprintf("%s=~\"%s\"", key, joinedValue))
	}

	return formattedCriteria
}

func createLokiBody(activity models.Activity) (LokiBody, error) {
	labels, metadata := lokiLabels(activity.Filter.Fields.ToMap())
	labels["service_name"] = configuration.AppName

	logLine := map[string]interface{}{
		"message": activity.Message,
	}

	if isAuthorizedObject(activity.Filter.Fields.ObjectType) {
		logLine["object"] = activity.Object
	}

	logLineJSON, err := json.Marshal(logLine)
	if err != nil {
		zap.L().Error("Failed to marshal log line", zap.Error(err))
		return LokiBody{}, err
	}

	entry := RawLogValue{activity.Filter.Timestamp, string(logLineJSON), metadata}
	stream := StreamEntry{
		Stream: labels,
		Values: []RawLogValue{entry},
	}

	return LokiBody{
		Streams: []StreamEntry{stream},
	}, nil
}

var lokiLabelSet = func() map[string]bool {
	m := map[string]bool{}
	t := reflect.TypeOf(models.ActivityFields{})
	for i := range t.NumField() {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		if lokiTag := f.Tag.Get("loki"); lokiTag == "label" {
			m[jsonTag] = true
		}
	}
	return m
}()

func lokiLabels[T any](fields map[string]T) (map[string]T, map[string]T) {
	labels := make(map[string]T)
	metadata := make(map[string]T)
	for key, value := range fields {
		if lokiLabelSet[key] {
			labels[key] = value
		} else {
			metadata[key] = value
		}
	}
	return labels, metadata
}
