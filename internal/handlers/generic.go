package handlers

import (
	customErr "api/internal/errors"
	h "api/internal/helpers"
	"api/internal/models"
	"errors"
	"github.com/google/uuid"
	"net/http"
)

type CreateTargetFunc[In any, Out any] func(uuid.UUIDs, In) (Out, error)
type ListTargetFunc[Out any] func() []Out
type GetOneTargetFunc[Out any] func(uuid.UUIDs) (Out, error)
type UpdateTargetFunc[In any, Out any] func(uuid.UUIDs, In) (Out, error)
type DeleteTargetFunc func(uuid.UUIDs) error

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		resp, err := create(ids, r.Context().Value(h.BodyKey{}).(In))
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusBadRequest, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func GetListHandler[Out any](getList ListTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records := getList()
		page := models.Page[Out]{Data: records}
		h.RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandler[Out any](getOne GetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		record, err := getOne(ids)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func UpdateHandler[In any, Out any](update UpdateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		_, err := update(ids, r.Context().Value(h.BodyKey{}).(In))
		if err != nil {
			strErrors := []string{err.Error()}

			var apiErr *customErr.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, strErrors)
			} else {
				h.RespondWithError(w, http.StatusBadRequest, strErrors)
			}
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func DeleteHandler(delete DeleteTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		err := delete(ids)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
