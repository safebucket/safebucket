package handlers

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type (
	ShareGetOneTargetFunc[Out any]                 func(*zap.Logger, models.Share, uuid.UUIDs) (Out, error)
	ShareGetOneWithQueryTargetFunc[Q any, Out any] func(*zap.Logger, models.Share, uuid.UUIDs, Q) (Out, error)
	ShareCreateTargetFunc[In any, Out any]         func(*zap.Logger, models.Share, uuid.UUIDs, In) (Out, error)
	ShareActionTargetFunc                          func(*zap.Logger, models.Share, uuid.UUIDs) error
	ShareAuthTargetFunc[In any]                    func(bool, *zap.Logger, models.Share, uuid.UUIDs, In) (AuthFlowResult, error)
)

func getShare(r *http.Request) models.Share {
	share, _ := r.Context().Value(m.ShareKey{}).(models.Share)
	return share
}

func ShareAuthHandler[In any](forceSecure bool, auth ShareAuthTargetFunc[In]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), spanName(auth))
		defer span.End()
		r = r.WithContext(ctx)

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
		result, err := auth(isSecureRequest(r, forceSecure), logger, share, ids, body)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		writeCookies(w, result.Cookies)
		h.RespondWithJSON(w, result.Status, result.Body)
	}
}

func ShareGetOneHandler[Out any](getOne ShareGetOneTargetFunc[Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), spanName(getOne))
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)
		record, err := getOne(logger, share, ids)
		if err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func ShareGetOneWithQueryHandler[Q any, Out any](getOne ShareGetOneWithQueryTargetFunc[Q, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), spanName(getOne))
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)

		query, ok := r.Context().Value(models.QueryKey{}).(Q)
		if !ok {
			logger.Error("Failed to extract query params from context")
			WriteError(span, w, nil)
			return
		}

		record, err := getOne(logger, share, ids, query)
		if err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func ShareCreateHandler[In any, Out any](create ShareCreateTargetFunc[In, Out]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), spanName(create))
		defer span.End()
		r = r.WithContext(ctx)

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
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func ShareActionHandler(action ShareActionTargetFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), spanName(action))
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		share := getShare(r)
		logger := m.GetLogger(r)
		if err := action(logger, share, ids); err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
