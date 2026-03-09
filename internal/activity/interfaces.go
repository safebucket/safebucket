package activity

import "github.com/safebucket/safebucket/internal/models"

type IActivityLogger interface {
	Search(searchCriteria map[string][]string) ([]map[string]interface{}, error)
	Send(message models.Activity) error
	CountByDay(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error)
	Close() error
}
