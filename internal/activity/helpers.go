package activity

import (
	"reflect"
	"sort"
	"strconv"
	"time"

	"api/internal/models"

	"gorm.io/gorm"
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
		Timestamp: strconv.FormatInt(time.Now().UnixNano(), 10),
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
		ts1, ok1 := enrichedActivity[i]["timestamp"].(string)
		if !ok1 {
			return false
		}
		ts2, ok2 := enrichedActivity[j]["timestamp"].(string)
		if !ok2 {
			return true
		}

		t1, err1 := strconv.ParseInt(ts1, 10, 64)
		if err1 != nil {
			t1 = 0
		}
		t2, err2 := strconv.ParseInt(ts2, 10, 64)
		if err2 != nil {
			t2 = 0
		}
		return t1 > t2
	})

	return enrichedActivity
}
