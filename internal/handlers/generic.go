package handlers

import (
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type (
	CreateTargetFunc[In any, Out any]         func(*zap.Logger, models.UserClaims, uuid.UUIDs, In) (Out, error)
	ListTargetFunc[Out any]                   func(*zap.Logger, models.UserClaims, uuid.UUIDs) []Out
	GetOneTargetFunc[Out any]                 func(*zap.Logger, models.UserClaims, uuid.UUIDs) (Out, error)
	GetOneWithQueryTargetFunc[Q any, Out any] func(*zap.Logger, models.UserClaims, uuid.UUIDs, Q) (Out, error)
	GetOneListTargetFunc[Out any]             func(*zap.Logger, models.UserClaims, uuid.UUIDs) []Out
	BodyTargetFunc[In any]                    func(*zap.Logger, models.UserClaims, uuid.UUIDs, In) error
	DeleteTargetFunc                          func(*zap.Logger, models.UserClaims, uuid.UUIDs) error
)

func spanName(fn any) string {
	full := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := strings.Split(full, ".")
	if len(parts) < 2 {
		return full
	}
	method := strings.TrimSuffix(parts[len(parts)-1], "-fm")
	return parts[len(parts)-2] + "." + method
}

func recordError(span trace.Span, err error, status int) {
	span.RecordError(err)
	span.SetAttributes(attribute.Int("http.status_code", status))
	if status >= 500 {
		span.SetStatus(codes.Error, err.Error())
	}
}

func resolveError(err error) (int, string) {
	var apiErr *apierrors.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status, apiErr.Code
	}
	return http.StatusInternalServerError, apierrors.CodeInternalServerError
}

func WriteError(span trace.Span, w http.ResponseWriter, err error) {
	status, code := resolveError(err)
	recordError(span, err, status)
	h.RespondWithError(w, status, []string{code})
}

//nolint:dupl // structurally similar to GetOneWithQueryHandler but handles body decoding and write semantics.
func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	name := spanName(create)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}
		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			return
		}

		resp, err := create(logger, claims, ids, body)
		if err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func GetListHandler[Out any](getList ListTargetFunc[Out]) http.HandlerFunc {
	name := spanName(getList)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		records := getList(logger, claims, ids)
		page := models.Page[Out]{Data: records}
		h.RespondWithJSON(w, http.StatusOK, page)
	}
}

func GetOneHandler[Out any](getOne GetOneTargetFunc[Out]) http.HandlerFunc {
	name := spanName(getOne)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		record, err := getOne(logger, claims, ids)
		if err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

//nolint:dupl // structurally similar to CreateHandler but handles query decoding and read semantics.
func GetOneWithQueryHandler[Q any, Out any](getOne GetOneWithQueryTargetFunc[Q, Out]) http.HandlerFunc {
	name := spanName(getOne)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		query, ok := r.Context().Value(models.QueryKey{}).(Q)
		if !ok {
			logger.Error("Failed to extract query params from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			return
		}

		record, err := getOne(logger, claims, ids, query)
		if err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func BodyHandler[In any](handler BodyTargetFunc[In]) http.HandlerFunc {
	name := spanName(handler)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{apierrors.CodeInternalServerError})
			return
		}

		if err := handler(logger, claims, ids, body); err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}

func DeleteHandler(del DeleteTargetFunc) http.HandlerFunc {
	name := spanName(del)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		if err := del(logger, claims, ids); err != nil {
			WriteError(span, w, err)
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
