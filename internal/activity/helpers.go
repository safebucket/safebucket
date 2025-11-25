package activity

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"time"

	"api/internal/models"

	"github.com/google/uuid"
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
	"folder_id": {Name: "folder", Object: models.Folder{}},
}

// NewLogFilter creates a LogFilter object with the specified criteria and the current timestamp in nanoseconds.
func NewLogFilter(criteria map[string]string) models.LogFilter {
	return models.LogFilter{
		Fields:    criteria,
		Timestamp: strconv.FormatInt(time.Now().UnixNano(), 10),
	}
}

// EnrichActivity returns a new slice of logs with specified fields enriched by fetching related objects.
// It uses a three-tier lookup strategy:
// 1. Use object from Loki metadata (if present)
// 2. Use cached DB result (if already queried)
// 3. Query DB and cache the result
// It does not mutate the original `activity` slice.
func EnrichActivity(db *gorm.DB, activity []map[string]interface{}) []map[string]interface{} {
	enrichedActivity := make([]map[string]interface{}, 0, len(activity))

	// Initialize unified DB query cache (only for DB results, NOT Loki data)
	cache := make(map[uuid.UUID]interface{})

	for _, log := range activity {
		newLog := log

		// Tier 1: Check if object exists in Loki metadata
		objectType, _ := log["object_type"].(string)
		if objectData, ok := log["object"]; ok && objectData != nil {
			if objectMap, ok := objectData.(map[string]interface{}); ok {
				jsonBytes, _ := json.Marshal(objectMap)

				switch objectType {
				case "bucket":
					var bucket models.Bucket
					if json.Unmarshal(jsonBytes, &bucket) == nil {
						newLog["bucket"] = &bucket
						delete(newLog, "bucket_id")
					}
				case "file":
					var file models.File
					if json.Unmarshal(jsonBytes, &file) == nil {
						newLog["file"] = &file
						delete(newLog, "file_id")
					}
				case "folder":
					var folder models.Folder
					if json.Unmarshal(jsonBytes, &folder) == nil {
						newLog["folder"] = &folder
						delete(newLog, "folder_id")
					}
				}
				delete(newLog, "object")
			}
		}

		// Tier 2 & 3: For remaining ID fields, check cache then DB
		for fieldName, enrichedField := range ToEnrich {
			if val, ok := log[fieldName]; ok && val != "" {
				if _, alreadyEnriched := newLog[enrichedField.Name]; alreadyEnriched {
					continue
				}

				idStr, ok := val.(string)
				if !ok {
					continue
				}

				id, err := uuid.Parse(idStr)
				if err != nil {
					continue
				}

				var object interface{}
				if cached, exists := cache[id]; exists {
					object = cached
				} else {
					object = reflect.New(reflect.TypeOf(enrichedField.Object)).Interface()
					db.Unscoped().Where("id = ?", id).First(object)
					cache[id] = object
				}

				newLog[enrichedField.Name] = object
				delete(newLog, fieldName)
			}
		}

		enrichedActivity = append(enrichedActivity, newLog)
	}

	return sortByTimestamp(enrichedActivity)
}

func sortByTimestamp(activity []map[string]interface{}) []map[string]interface{} {
	sort.Slice(activity, func(i, j int) bool {
		ts1, ok1 := activity[i]["timestamp"].(string)
		if !ok1 {
			return false
		}
		ts2, ok2 := activity[j]["timestamp"].(string)
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

	return activity
}
