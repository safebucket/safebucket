package helpers

import (
	"context"
	"errors"

	"api/internal/models"
)

// GetQueryParams extracts validated query parameters from the request context.
// It returns an error if the query parameters are not found or cannot be type-asserted.
//
// Usage:
//
//	queryParams, err := h.GetQueryParams[models.FileListQueryParams](r.Context())
//	if err != nil {
//	    logger.Error("Failed to extract query params from context")
//	    h.RespondWithError(w, http.StatusInternalServerError, []string{"INTERNAL_SERVER_ERROR"})
//	    return
//	}
func GetQueryParams[T any](c context.Context) (T, error) {
	value, ok := c.Value(models.QueryKey{}).(T)
	if !ok {
		var zero T
		return zero, errors.New("invalid query params")
	}
	return value, nil
}
