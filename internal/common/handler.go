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

func CreateHandler[T any](repo GenericRepo[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := repo.Create(r.Context().Value("body").(T))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusBadRequest, strErrors)
		} else {
			RespondWithJSON(w, http.StatusCreated, nil)
		}
	}
}

func UpdateHandler[T any](repo GenericRepo[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		// FIXME(YLB): Converting the body struct (used by validation) to JSON
		// FIXME(YLB): Then converting back to the actual DB struct to perform DB operations
		body := r.Context().Value("body")
		jsonString, _ := json.Marshal(body)
		record := new(T)
		json.Unmarshal(jsonString, &record)

		_, err = repo.Update(uint(id), *record)
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func GetListHandler[T any](repo GenericRepo[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records := repo.GetList()
		page := Page[T]{Data: records}
		RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandler[T any](repo GenericRepo[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		record, err := repo.GetOne(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func DeleteHandler[T any](repo GenericRepo[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		err = repo.Delete(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

type CreateTargetFunc[In any] func(In) error
type ListTargetFunc[Out any] func() []Out
type GetOneTargetFunc[Out any] func(id uint) (Out, error)
type UpdateTargetFunc[In any, Out any] func(id uint, body In) (Out, error)
type DeleteTargetFunc func(id uint) error

func CreateHandlerV2[In any](f CreateTargetFunc[In]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(r.Context().Value("body").(In))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusBadRequest, strErrors)
		} else {
			RespondWithJSON(w, http.StatusCreated, nil)
		}
	}
}

func GetListHandlerV2[Out any](f ListTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records := f()
		page := Page[Out]{Data: records}
		RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandlerV2[Out any](f GetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		record, err := f(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func UpdateHandlerV2[In any, Out any](f UpdateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		_, err = f(uint(id), r.Context().Value("body").(In))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func DeleteHandlerV2(f DeleteTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}

		err = f(uint(id))
		if err != nil {
			strErrors := []string{err.Error()}
			RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
