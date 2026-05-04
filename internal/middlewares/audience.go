package middlewares

import (
	"net/http"
	"slices"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"
)

func AudienceValidate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), "middleware.AudienceValidate")
		defer span.End()
		r = r.WithContext(ctx)

		if excluded, _ := r.Context().Value(AuthExcludedKey{}).(bool); excluded {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		if !ok {
			helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
			return
		}

		tokenAudience := claims.AudienceString()

		allowedAudiences := getRouteAllowedAudiences(r.URL.Path, r.Method)

		if allowedAudiences != nil {
			if !isAudienceInList(tokenAudience, allowedAudiences) {
				helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}
		} else {
			if tokenAudience != configuration.AudienceAccessToken {
				helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func getRouteAllowedAudiences(path, method string) []string {
	for _, rule := range configuration.AuthAudienceRules {
		if rule.Method != "*" && rule.Method != method {
			continue
		}

		if (rule.ExactPath != "" && rule.ExactPath == path) || (rule.Pattern != nil && rule.Pattern.MatchString(path)) {
			return rule.AllowedAudiences
		}
	}
	return nil
}

func isAudienceInList(audience string, allowedAudiences []string) bool {
	return slices.Contains(allowedAudiences, audience)
}

func isAudienceAllowedForRoute(audience, path, method string) bool {
	allowedAudiences := getRouteAllowedAudiences(path, method)
	if allowedAudiences == nil {
		return false
	}
	return isAudienceInList(audience, allowedAudiences)
}
