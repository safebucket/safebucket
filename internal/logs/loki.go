package logs

import (
	"api/internal/models"
	"fmt"
	"github.com/go-resty/resty/v2"
	"time"
)

var authorizedLabels = [3]string{"domain", "obj_type", "action"}

const lokiPushURI = "/loki/api/v1/push"
const lokiSearchURI = "loki/api/v1/query"

type LokiBody struct {
	Streams []StreamEntry `json:"streams"`
}

type StreamEntry struct {
	Stream map[string]string `json:"stream"` // dynamic labels like "foo": "bar2"
	Values []RawLogValue     `json:"values"` // each entry is a [timestamp, message]
}
type RawLogValue [3]interface{}

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
		SetRetryMaxWaitTime(20 * time.Second)

	// TODO: Retry conditions mechanism

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(lokiBody).
		Post(s.pushURL)

	if err != nil {
		fmt.Println("Erreur lors de la requête:", err)
		return nil
	}
	fmt.Println("Statut HTTP :", resp.Status())

	if resp.StatusCode() != 204 {
		return fmt.Errorf("loki response status code %d", resp.StatusCode())
	}

	return nil
}

func (s *LokiClient) Search(key map[string]string) error {
	panic("implement me")
}

func isAuthorized(label string) bool {
	for _, auth := range authorizedLabels {
		if label == auth {
			return true
		}
	}
	return false
}

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

func createLokiBody(log models.LogMessage) (LokiBody, error) {
	labels, metadata := splitMetadata(log.SearchCriteria)
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	entry := RawLogValue{timestamp, log.Message, metadata}
	stream := StreamEntry{
		Stream: labels,
		Values: []RawLogValue{entry},
	}
	body := LokiBody{
		Streams: []StreamEntry{stream},
	}
	return body, nil
}

func NewLokiClient(config models.LogConfiguration) IClient {
	return &LokiClient{
		Endpoint:  config.Loki.Endpoint,
		pushURL:   fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiPushURI),
		searchURL: fmt.Sprintf("%s%s", config.Loki.Endpoint, lokiSearchURI),
	}
}
