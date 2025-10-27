package handlers

import (
	"errors"
	"net/http"

	customErr "api/internal/errors"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type (
	CreateTargetFunc[In any, Out any] func(*zap.Logger, models.UserClaims, uuid.UUIDs, In) (Out, error)
	ListTargetFunc[Out any]           func(*zap.Logger, models.UserClaims, uuid.UUIDs) []Out
	GetOneTargetFunc[Out any]         func(*zap.Logger, models.UserClaims, uuid.UUIDs) (Out, error)
	GetOneListTargetFunc[Out any]     func(*zap.Logger, models.UserClaims, uuid.UUIDs) []Out
	UpdateTargetFunc[In any]          func(*zap.Logger, models.UserClaims, uuid.UUIDs, In) error
	DeleteTargetFunc                  func(*zap.Logger, models.UserClaims, uuid.UUIDs) error
)

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}
		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		resp, err := create(logger, claims, ids, r.Context().Value(m.BodyKey{}).(In))
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
		logger := m.GetLogger(r)
		records := getList(logger, claims, ids)
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
		logger := m.GetLogger(r)
		record, err := getOne(logger, claims, ids)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func UpdateHandler[In any](update UpdateTargetFunc[In]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		err := update(logger, claims, ids, r.Context().Value(m.BodyKey{}).(In))
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
		logger := m.GetLogger(r)
		err := delete(logger, claims, ids)
		if err != nil {
			strErrors := []string{err.Error()}
			h.RespondWithError(w, http.StatusNotFound, strErrors)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
