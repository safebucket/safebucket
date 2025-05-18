package activity

import "api/internal/models"

// IActivityLogger defines a common interface for all logs
type IActivityLogger interface {
	Search(searchCriteria map[string][]string) ([]models.History, error)
	Send(message models.LogMessage) error
}
