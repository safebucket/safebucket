package logs

import "api/internal/models"

// ILogClient defines a common interface for all logs
type ILogClient interface {
	Search(searchCriteria map[string][]string) ([]models.History, error)
	Send(message models.LogMessage) error
}
