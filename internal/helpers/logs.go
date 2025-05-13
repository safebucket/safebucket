package helpers

import (
	"api/internal/models"
	"fmt"
	"time"
)

// NewSearchCriteria creates a LogFilter object with the specified criterias and the current timestamp in nanoseconds.
func NewSearchCriteria(criterias map[string]string) models.LogFilter {
	return models.LogFilter{
		Fields:    criterias,
		Timestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}
