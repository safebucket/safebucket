package helpers

import (
	"api/internal/models"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

const maxUuids = 2

func ParseUUIDs(w http.ResponseWriter, r *http.Request) (uuid.UUIDs, bool) {
	// Hard limit for maximum UUIDs in the URL to avoid unexpected behaviours
	var ids uuid.UUIDs

	for i := range maxUuids {
		idStr := chi.URLParam(r, fmt.Sprintf("id%d", i))
		if idStr == "" {
			return ids, true
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			strErrors := []string{"INVALID_UUID"}
			RespondWithError(w, http.StatusBadRequest, strErrors)
			return ids, false
		}

		ids = append(ids, id)
	}

	return ids, true
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	if payload != nil {
		response, err := json.Marshal(payload)
		if err != nil {
			zap.L().Error("Failed to encode to JSON", zap.Error(err))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(code)
		_, err = w.Write(response)
		if err != nil {
			zap.L().Error("Failed to write response", zap.Error(err))
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(code)
	}
}

func RespondWithError(w http.ResponseWriter, code int, msg []string) {
	RespondWithJSON(w, code, models.Error{Status: code, Error: msg})
}
