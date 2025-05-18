package helpers

import (
	"api/internal/models"
	"fmt"
	"time"
)

// NewLogFilter creates a LogFilter object with the specified criteria and the current timestamp in nanoseconds.
func NewLogFilter(criteria map[string]string) models.LogFilter {
	return models.LogFilter{
		Fields:    criteria,
		Timestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}
