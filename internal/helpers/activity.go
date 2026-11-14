package helpers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultActivityWindowDays = 30
	maxActivityWindowDays     = 90
	defaultActivityLimit      = 100
)

func ResolveActivityRange(from, to time.Time) (time.Time, time.Time, error) {
	end := time.Now().UTC()
	if !to.IsZero() {
		end = to.UTC()
	}

	start := end.AddDate(0, 0, -defaultActivityWindowDays)
	if !from.IsZero() {
		start = from.UTC()
	}

	if start.After(end) {
		return time.Time{}, time.Time{}, apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
	}

	if end.Sub(start) > maxActivityWindowDays*24*time.Hour {
		return time.Time{}, time.Time{}, apierrors.New(http.StatusBadRequest, apierrors.CodeActivityRangeTooLarge)
	}

	return start, end, nil
}

func ParseActivityCursor(cursor string) (time.Time, error) {
	nanos, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil {
		return time.Time{}, apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
	}
	return time.Unix(0, nanos).UTC(), nil
}

func PaginateActivity(rows []map[string]interface{}, limit int) ([]map[string]interface{}, *string) {
	if len(rows) <= limit {
		return rows, nil
	}

	rows = rows[:limit]
	ts, ok := rows[len(rows)-1]["timestamp"].(string)
	if !ok {
		return rows, nil
	}

	return rows, &ts
}

func SearchActivityPage(
	db *gorm.DB,
	logger *zap.Logger,
	activityLogger activity.IActivityLogger,
	criteria map[string][]string,
	params models.ActivityQueryParams,
) (models.Page[map[string]interface{}], error) {
	empty := models.Page[map[string]interface{}]{Data: []map[string]interface{}{}}

	start, end, err := ResolveActivityRange(params.From, params.To)
	if err != nil {
		return models.Page[map[string]interface{}]{}, err
	}

	if params.Cursor != "" {
		cursorTime, cursorErr := ParseActivityCursor(params.Cursor)
		if cursorErr != nil {
			return models.Page[map[string]interface{}]{}, cursorErr
		}
		end = cursorTime
	}

	limit := params.Limit
	if params.Limit <= 0 {
		limit = defaultActivityLimit
	}

	rows, err := activityLogger.Search(criteria, start, end, limit)
	if err != nil {
		logger.Error("Search history failed", zap.Error(err))
		return models.Page[map[string]interface{}]{}, err
	}

	data, nextCursor := PaginateActivity(rows, limit)
	if len(data) == 0 {
		return empty, nil
	}

	return models.Page[map[string]interface{}]{
		Data:       activity.EnrichActivity(db, data),
		NextCursor: nextCursor,
	}, nil
}
