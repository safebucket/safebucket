package activity

import "api/internal/models"

// IActivityLogger defines a common interface for all logs.
type IActivityLogger interface {
	Search(searchCriteria map[string][]string) ([]map[string]interface{}, error)
	Send(message models.Activity) error
	CountByDay(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error)
	Close() error
}
