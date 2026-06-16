package activity

import (
	"time"

	"github.com/safebucket/safebucket/internal/models"
)

type IActivityLogger interface {
	Search(searchCriteria map[string][]string, start, end time.Time, limit int) ([]map[string]interface{}, error)
	Send(message models.Activity) error
	CountByHour(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error)
	Close() error
}
