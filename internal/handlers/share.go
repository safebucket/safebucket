package handlers

import (
	"errors"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type (
	ShareGetOneTargetFunc[Out any]         func(*zap.Logger, models.Share, uuid.UUIDs) (Out, error)
	ShareCreateTargetFunc[In any, Out any] func(*zap.Logger, models.Share, uuid.UUIDs, In) (Out, error)
	ShareActionTargetFunc                  func(*zap.Logger, models.Share, uuid.UUIDs) error
	ShareAuthTargetFunc[In any, Out any]   func(*zap.Logger, models.Share, uuid.UUIDs, In) (Out, error)
)

func getShare(r *http.Request) models.Share {
	share, _ := r.Context().Value(m.ShareKey{}).(models.Share)
	return share
}

func ShareAuthHandler[In any, Out any](auth ShareAuthTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			return
		}

		share := getShare(r)
		resp, err := auth(logger, share, ids, body)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{apiErr.Message})
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			}
		} else {
			h.RespondWithJSON(w, http.StatusOK, resp)
		}
	}
}

func ShareGetOneHandler[Out any](getOne ShareGetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)
		record, err := getOne(logger, share, ids)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{apiErr.Message})
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			}
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func ShareCreateHandler[In any, Out any](create ShareCreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			return
		}

		resp, err := create(logger, share, ids, body)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{apiErr.Message})
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			}
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func ShareActionHandler(action ShareActionTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)
		err := action(logger, share, ids)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{apiErr.Message})
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			}
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
