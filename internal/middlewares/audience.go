package middlewares

import (
	"net/http"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
)

// AudienceValidate middleware handles audience validation for JWT tokens.
// It validates that the token's audience claim is appropriate for the route.
// This middleware should be applied after Authenticate middleware.
//
// Logic:
// 1. Skip validation if auth was excluded
// 2. For routes with explicit audience rules (AuthAudienceRules), validate against those rules
// 3. For all other routes, require the full access token audience ("app:*").
func AudienceValidate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if auth was excluded (context set by Authenticate)
		if excluded, _ := r.Context().Value(AuthExcludedKey{}).(bool); excluded {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		if !ok {
			// No claims means auth middleware didn't set them (shouldn't happen if middleware order is correct)
			helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
			return
		}

		allowedAudiences := getRouteAllowedAudiences(r.URL.Path, r.Method)

		if allowedAudiences != nil {
			if !isAudienceInList(claims.Aud, allowedAudiences) {
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}
		} else {
			if claims.Aud != configuration.AudienceAccessToken {
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
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
	for _, allowed := range allowedAudiences {
		if audience == allowed {
			return true
		}
	}
	return false
}

// isAudienceAllowedForRoute checks if a token's audience is permitted for the route.
// Returns false if the route has no audience rules or if the audience is not in the allowed list.
// This function is primarily for testing and internal validation.
func isAudienceAllowedForRoute(audience, path, method string) bool {
	allowedAudiences := getRouteAllowedAudiences(path, method)
	if allowedAudiences == nil {
		return false // No rule = restricted tokens not allowed
	}
	return isAudienceInList(audience, allowedAudiences)
}
