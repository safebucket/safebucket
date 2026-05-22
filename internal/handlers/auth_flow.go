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

type AuthFlowResult struct {
	Status  int
	Body    any
	Cookies []*http.Cookie
}

type AuthFlowTargetFunc[In any] func(
	isSecure bool,
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body In,
) (AuthFlowResult, error)

func AuthFlowHandler[In any](forceSecure bool, target AuthFlowTargetFunc[In]) http.HandlerFunc {
	name := spanName(target)
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

		result, err := target(isSecureRequest(r, forceSecure), logger, claims, ids, body)
		if err != nil {
			WriteError(span, w, err)
			return
		}

		writeCookies(w, result.Cookies)
		h.RespondWithJSON(w, result.Status, result.Body)
	}
}
