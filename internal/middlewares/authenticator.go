package middlewares

import (
	"context"
	"net/http"
	"strings"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
)

// RestrictedAccessKey is used to store restricted access flag in context.
type RestrictedAccessKey struct{}

func Authenticate(jwtSecret string, mfaRequired bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			accessToken := r.Header.Get("Authorization")

			// Try parsing as full access token first
			userClaims, err := helpers.ParseAccessToken(jwtSecret, accessToken)
			if err == nil {
				// Full access token - proceed normally
				if mfaRequired && !userClaims.MFA && userClaims.Provider == string(models.LocalProviderType) {
					if !isMFABypassPath(r.URL.Path, r.Method) {
						helpers.RespondWithError(w, 403, []string{"MFA_SETUP_REQUIRED"})
						return
					}
				}
				ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try parsing as restricted access token
			userClaims, err = helpers.ParseRestrictedAccessToken(jwtSecret, accessToken)
			if err == nil {
				// Validate audience against route-specific rules
				// This is the single validation point - handlers don't need audience checks
				if !isAudienceAllowedForRoute(userClaims.Aud, r.URL.Path, r.Method) {
					helpers.RespondWithError(w, 403, []string{"INVALID_TOKEN_FOR_ROUTE"})
					return
				}
				ctx := context.WithValue(r.Context(), models.UserClaimKey{}, userClaims)
				ctx = context.WithValue(ctx, RestrictedAccessKey{}, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No valid token found
			if isMFABypassPath(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
		}
		return http.HandlerFunc(fn)
	}
}

// routeAudienceRule defines which token audiences are allowed for a specific route.
// This centralizes token validation and eliminates the need for handler-level audience checks.
type routeAudienceRule struct {
	pathPrefix       string
	pathSuffix       string
	method           string
	allowedAudiences []string
}

// routeAudienceRules defines the security policy for restricted token access.
// Routes not listed here will reject restricted tokens entirely.
// This is the single source of truth for which audiences can access which endpoints.
var routeAudienceRules = []routeAudienceRule{
	// MFA verification - accepts BOTH login and password reset tokens
	// VerifyMFALogin handles differential behavior based on audience
	{
		pathPrefix:       "/api/v1/auth/mfa/verify",
		pathSuffix:       "",
		method:           "POST",
		allowedAudiences: []string{configuration.AudienceMFALogin, configuration.AudienceMFAReset},
	},

	// Password reset completion - ONLY password reset tokens allowed
	// This prevents cross-flow attacks from login tokens
	{
		pathPrefix:       "/api/v1/auth/reset-password/",
		pathSuffix:       "/complete",
		method:           "POST",
		allowedAudiences: []string{configuration.AudienceMFAReset},
	},

	// MFA device listing - both token types for setup flow
	{
		pathPrefix:       "/api/v1/users/",
		pathSuffix:       "/mfa/devices",
		method:           "GET",
		allowedAudiences: []string{configuration.AudienceMFALogin, configuration.AudienceMFAReset},
	},

	// MFA device creation - both token types for initial setup
	{
		pathPrefix:       "/api/v1/users/",
		pathSuffix:       "/mfa/devices",
		method:           "POST",
		allowedAudiences: []string{configuration.AudienceMFALogin, configuration.AudienceMFAReset},
	},

	// MFA device verification - both token types
	{
		pathPrefix:       "/api/v1/users/",
		pathSuffix:       "/verify",
		method:           "POST",
		allowedAudiences: []string{configuration.AudienceMFALogin, configuration.AudienceMFAReset},
	},
}

// getRouteAllowedAudiences returns the allowed audiences for a route.
// Returns nil if no restricted token rule exists (route requires full access token).
func getRouteAllowedAudiences(path, method string) []string {
	for _, rule := range routeAudienceRules {
		if rule.method != method {
			continue
		}
		if rule.pathSuffix == "" {
			if strings.HasPrefix(path, rule.pathPrefix) {
				return rule.allowedAudiences
			}
		} else {
			if strings.HasPrefix(path, rule.pathPrefix) && strings.HasSuffix(path, rule.pathSuffix) {
				return rule.allowedAudiences
			}
		}
	}
	return nil
}

// isAudienceAllowedForRoute checks if a token's audience is permitted for the route.
// Returns false if the route has no audience rules or if the audience is not in the allowed list.
func isAudienceAllowedForRoute(audience, path, method string) bool {
	allowedAudiences := getRouteAllowedAudiences(path, method)
	if allowedAudiences == nil {
		return false // No rule = restricted tokens not allowed
	}
	for _, allowed := range allowedAudiences {
		if audience == allowed {
			return true
		}
	}
	return false
}

func isMFABypassPath(path, method string) bool {
	for _, rule := range configuration.MFABypassRules {
		if rule.Method != "*" && rule.Method != method {
			continue
		}

		if rule.PathSuffix == "" {
			if strings.HasPrefix(path, rule.PathPrefix) {
				remaining := strings.TrimPrefix(path, rule.PathPrefix)
				if remaining == "" || !strings.Contains(remaining, "/") {
					return true
				}
			}
		} else {
			if strings.HasPrefix(path, rule.PathPrefix) && strings.HasSuffix(path, rule.PathSuffix) {
				return true
			}
		}
	}
	return false
}

func isExcluded(path, method string) bool {
	if exactRules, exists := configuration.AuthRuleExactMatchPath[path]; exists {
		for _, rule := range exactRules {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth
			}
		}
	}

	for _, rule := range configuration.AuthRulePrefixMatchPath {
		if strings.HasPrefix(path, rule.Path) {
			if rule.Method == "*" || rule.Method == method {
				return !rule.RequireAuth
			}
		}
	}

	return false
}
