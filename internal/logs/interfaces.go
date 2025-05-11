package logs

import "api/internal/models"

// IClient defines a common interface for all logs
type IClient interface {
	Search(keys map[string]string) error
	Send(message models.LogMessage) error
}
