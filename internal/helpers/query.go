package helpers

import (
	"context"
	"errors"

	"github.com/safebucket/safebucket/internal/models"
)

func GetQueryParams[T any](c context.Context) (T, error) {
	value, ok := c.Value(models.QueryKey{}).(T)
	if !ok {
		var zero T
		return zero, errors.New("invalid query params")
	}
	return value, nil
}
