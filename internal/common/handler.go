package common

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"strconv"
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
	RespondWithJSON(w, code, Error{Status: code, Error: msg})
}

type CreateTargetFunc[In any] func(In) error
type ListTargetFunc[Out any] func() []Out
type GetOneTargetFunc[Out any] func(uint) (Out, error)
type UpdateTargetFunc[In any, Out any] func(uint, In) (Out, error)
type DeleteTargetFunc func(uint) error

func CreateHandler[In any](create CreateTargetFunc[In]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := create(r.Context().Value("body").(In))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusBadRequest, strErrors)
		} else {
			RespondWithJSON(w, http.StatusCreated, nil)
		}
	}
}

func GetListHandler[Out any](getList ListTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records := getList()
		page := Page[Out]{Data: records}
		RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandler[Out any](getOne GetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		record, err := getOne(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func UpdateHandler[In any, Out any](update UpdateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		_, err = update(uint(id), r.Context().Value("body").(In))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func DeleteHandler(delete DeleteTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		err = delete(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
