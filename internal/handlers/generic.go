package handlers

import (
	customErr "api/internal/errors"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"errors"
	"github.com/google/uuid"
	"net/http"
)

type CreateTargetFunc[In any, Out any] func(models.UserClaims, uuid.UUIDs, In) (Out, error)
type ListTargetFunc[Out any] func(models.UserClaims, uuid.UUIDs) []Out
type GetOneTargetFunc[Out any] func(models.UserClaims, uuid.UUIDs) (Out, error)
type GetOneListTargetFunc[Out any] func(models.UserClaims, uuid.UUIDs) []Out
type UpdateTargetFunc[In any, Out any] func(models.UserClaims, uuid.UUIDs, In) (Out, error)
type DeleteTargetFunc func(models.UserClaims, uuid.UUIDs) error

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}
		claims, _ := h.GetUserClaims(r.Context())
		resp, err := create(claims, ids, r.Context().Value(m.BodyKey{}).(In))
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
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context()) // todo: check error
		records := getList(claims, ids)
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

		claims, _ := h.GetUserClaims(r.Context())
		record, err := getOne(claims, ids)
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

		claims, _ := h.GetUserClaims(r.Context())
		_, err := update(claims, ids, r.Context().Value(m.BodyKey{}).(In))
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

		claims, _ := h.GetUserClaims(r.Context())
		err := delete(claims, ids)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
