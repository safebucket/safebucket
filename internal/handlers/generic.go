package handlers

import (
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
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

// spanName extracts a readable "ServiceName.methodName" from a function pointer via reflection.
func spanName(fn any) string {
	full := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := strings.Split(full, ".")
	if len(parts) < 2 {
		return full
	}
	method := strings.TrimSuffix(parts[len(parts)-1], "-fm")
	return parts[len(parts)-2] + "." + method
}

func recordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func CreateHandler[In any, Out any](create CreateTargetFunc[In, Out]) http.HandlerFunc {
	name := spanName(create)
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}
		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{"INTERNAL_SERVER_ERROR"})
			return
		}

		resp, err := create(logger, claims, ids, body)
		if err != nil {
			recordError(span, err)
			h.RespondWithError(w, http.StatusBadRequest, []string{err.Error()})
		} else {
			h.RespondWithJSON(w, http.StatusCreated, resp)
		}
	}
}

func GetListHandler[Out any](getList ListTargetFunc[Out]) http.HandlerFunc {
	name := spanName(getList)
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

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
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		record, err := getOne(logger, claims, ids)
		if err != nil {
			recordError(span, err)
			h.RespondWithError(w, http.StatusNotFound, []string{err.Error()})
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func GetOneWithQueryHandler[Q any, Out any](getOne GetOneWithQueryTargetFunc[Q, Out]) http.HandlerFunc {
	name := spanName(getOne)
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		query, ok := r.Context().Value(models.QueryKey{}).(Q)
		if !ok {
			logger.Error("Failed to extract query params from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{"INTERNAL_SERVER_ERROR"})
			return
		}

		record, err := getOne(logger, claims, ids, query)
		if err != nil {
			recordError(span, err)
			h.RespondWithError(w, http.StatusNotFound, []string{err.Error()})
		} else {
			h.RespondWithJSON(w, http.StatusOK, record)
		}
	}
}

func BodyHandler[In any](handler BodyTargetFunc[In]) http.HandlerFunc {
	name := spanName(handler)
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)

		body, ok := r.Context().Value(m.BodyKey{}).(In)
		if !ok {
			logger.Error("Failed to extract body from context")
			h.RespondWithError(w, http.StatusInternalServerError, []string{"INTERNAL_SERVER_ERROR"})
			return
		}

		err := handler(logger, claims, ids, body)
		if err != nil {
			recordError(span, err)
			strErrors := []string{err.Error()}

			var apiErr *apierrors.APIError
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

func DeleteHandler(del DeleteTargetFunc) http.HandlerFunc {
	name := spanName(del)
	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(configuration.AppName).Start(r.Context(), name)
		defer span.End()

		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		claims, _ := h.GetUserClaims(r.Context())
		logger := m.GetLogger(r)
		err := del(logger, claims, ids)
		if err != nil {
			recordError(span, err)
			h.RespondWithError(w, http.StatusNotFound, []string{err.Error()})
		} else {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
		}
	}
}
