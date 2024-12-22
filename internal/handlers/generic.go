package handlers

import (
	h "api/internal/helpers"
	"api/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

type CreateTargetFunc[In any, Out any] func(*models.UserClaims, In) (Out, error)
type ListTargetFunc[Out any] func(*models.UserClaims) []Out
type GetOneTargetFunc[Out any] func(*models.UserClaims, uuid.UUID) (Out, error)
type UpdateTargetFunc[In any, Out any] func(*models.UserClaims, uuid.UUID, In) (Out, error)
type DeleteTargetFunc func(*models.UserClaims, uuid.UUID) error

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := h.GetUserClaims(r.Context())
		resp, err := create(claims, r.Context().Value(h.BodyKey{}).(In))
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusBadRequest, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func GetListHandler[Out any](getList func(u *models.UserClaims) []Out) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := h.GetUserClaims(r.Context())
		records := getList(claims)
		page := models.Page[Out]{Data: records}
		h.RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandler[Out any](getOne GetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := h.GetUserClaims(r.Context())
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}
		record, err := getOne(claims, id)
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
		claims, _ := h.GetUserClaims(r.Context())
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}
		_, err = update(claims, id, r.Context().Value(h.BodyKey{}).(In))
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
		claims, _ := h.GetUserClaims(r.Context())
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}
		err = delete(claims, id)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
