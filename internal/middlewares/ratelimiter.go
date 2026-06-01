package middlewares

import (
	"net/http"
	"strconv"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func applyRateLimit(
	next http.Handler,
	w http.ResponseWriter,
	r *http.Request,
	c cache.ICache,
	userIdentifier string,
	requestsPerMinute int,
) {
	retryAfter, err := cache.GetRateLimit(c, userIdentifier, requestsPerMinute)

	if err != nil {
		zap.L().Error("error", zap.Error(err))
		helpers.RespondWithError(w, 500, []string{apierrors.CodeInternalServerError})
		return
	}

	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		helpers.RespondWithError(w, http.StatusTooManyRequests, []string{apierrors.CodeRateLimitExceeded})
		return
	}

	next.ServeHTTP(w, r)
}

func RateLimit(
	cache cache.ICache,
	authenticatedRequestsPerMinute int,
	unauthenticatedRequestsPerMinute int,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.RateLimit")
			defer span.End()
			r = r.WithContext(ctx)

			claims, err := helpers.GetUserClaims(r.Context())
			if err != nil {
				info, ok := r.Context().Value(models.ClientInfoKey{}).(models.ClientInfo)
				if !ok || info.IP == "" {
					zap.L().Error("error", zap.Error(err))
					helpers.RespondWithErrorCtx(r.Context(), w, 500, []string{apierrors.CodeInternalServerError})
					return
				}
				applyRateLimit(next, w, r, cache, info.IP, unauthenticatedRequestsPerMinute)
			} else {
				userID := claims.UserID.String()
				applyRateLimit(next, w, r, cache, userID, authenticatedRequestsPerMinute)
			}
		}
		return http.HandlerFunc(fn)
	}
}
