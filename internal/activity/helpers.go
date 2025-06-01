package activity

import (
	"api/internal/models"
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"sort"
	"strconv"
	"time"
)

type ToEnrichValue struct {
	Name   string
	Object interface{}
}

var ToEnrich = map[string]ToEnrichValue{
	"user_id":   {Name: "user", Object: models.User{}},
	"bucket_id": {Name: "bucket", Object: models.Bucket{}},
	"file_id":   {Name: "file", Object: models.File{}},
}

// NewLogFilter creates a LogFilter object with the specified criteria and the current timestamp in nanoseconds.
func NewLogFilter(criteria map[string]string) models.LogFilter {
	return models.LogFilter{
		Fields:    criteria,
		Timestamp: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}

// EnrichActivity returns a new slice of logs with specified fields enriched by fetching related objects from the DB.
// It does not mutate the original `activity` slice.
func EnrichActivity(db *gorm.DB, activity []map[string]interface{}) []map[string]interface{} {
	enrichedActivity := make([]map[string]interface{}, 0, len(activity))

	for _, log := range activity {
		newLog := make(map[string]interface{})
		for k, v := range log {
			newLog[k] = v
		}

		for fieldName, enrichedField := range ToEnrich {
			if val, ok := log[fieldName]; ok && val != "" {
				object := reflect.New(reflect.TypeOf(enrichedField.Object)).Interface()

				db.Unscoped().Where("id = ?", val).First(object)

				newLog[enrichedField.Name] = object
				delete(newLog, fieldName)
			}
		}

		enrichedActivity = append(enrichedActivity, newLog)
	}

	sort.Slice(enrichedActivity, func(i, j int) bool {
		t1, _ := strconv.ParseInt(enrichedActivity[i]["timestamp"].(string), 10, 64)
		t2, _ := strconv.ParseInt(enrichedActivity[j]["timestamp"].(string), 10, 64)
		return t1 > t2
	})

	return enrichedActivity
}
