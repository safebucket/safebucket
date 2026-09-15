package models

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
)

type OptionalID struct {
	ID  *uuid.UUID
	Set bool
}

func (o *OptionalID) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}

	var id uuid.UUID
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	o.ID = &id

	return nil
}
