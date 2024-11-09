package handlers

import (
	h "api/internal/helpers"
	"api/internal/models"
	"github.com/google/uuid"
	"net/http"
)

type CreateTargetFunc[In any, Out any] func(In) (Out, error)
type ListTargetFunc[Out any] func() []Out
type GetOneTargetFunc[Out any] func(uuid.UUID) (Out, error)
type UpdateTargetFunc[In any, Out any] func(uuid.UUID, In) (Out, error)
type DeleteTargetFunc func(uuid.UUID) error

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := create(r.Context().Value(h.BodyKey{}).(In))
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
		id, ok := h.ParseUUID(w, r)
		if !ok {
			return
		}

		record, err := getOne(id)
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
		id, ok := h.ParseUUID(w, r)
		if !ok {
			return
		}

		_, err := update(id, r.Context().Value(h.BodyKey{}).(In))
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func DeleteHandler(delete DeleteTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.ParseUUID(w, r)
		if !ok {
			return
		}

		err := delete(id)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
