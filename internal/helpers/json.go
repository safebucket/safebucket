package helpers

import (
	"api/internal/models"
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
)

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
	}
}

func RespondWithError(w http.ResponseWriter, code int, msg []string) {
	RespondWithJSON(w, code, models.Error{Status: code, Error: msg})
}
