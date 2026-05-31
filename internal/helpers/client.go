package helpers

import (
	"context"
	"errors"

	"github.com/safebucket/safebucket/internal/models"
)

func GetClientInfo(c context.Context) (models.ClientInfo, error) {
	value, ok := c.Value(models.ClientInfoKey{}).(models.ClientInfo)
	if !ok {
		return models.ClientInfo{}, errors.New("invalid client info")
	}
	return value, nil
}
