package activity

import (
	"errors"
	"time"

	"github.com/safebucket/safebucket/internal/models"
)

var ErrRejected = errors.New("activity rejected by the backend")

type IActivityLogger interface {
	Search(searchCriteria map[string][]string, start, end time.Time, limit int) ([]map[string]interface{}, error)
	Send(activities ...models.Activity) error
	CountByHour(searchCriteria map[string][]string, days int) ([]models.TimeSeriesPoint, error)
	Close() error
}
