package middlewares

import (
	"net/http"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MFAValidate middleware handles MFA enforcement for routes that require it.
// This middleware should be applied after Authenticate and AudienceValidate middlewares.
//
// It checks:
// - If MFA is required by admin configuration and user has local provider
// - If user has MFA enabled (claims.MFA == true)
// - If user has enrolled MFA devices in DB (stale token detection)
// - If the route is in the MFA bypass list
//
// Note: Audience validation (restricted vs full access tokens) is handled by
// AudienceValidate middleware, not here.
func MFAValidate(db *gorm.DB, mfaRequired bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
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

			// Only enforce MFA for full access tokens from local provider
			if claims.Aud != configuration.AudienceAccessToken ||
				claims.Provider != string(models.LocalProviderType) {
				next.ServeHTTP(w, r)
				return
			}

			if claims.MFA {
				next.ServeHTTP(w, r)
				return
			}

			if isMFABypassPath(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if mfaRequired {
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			if db != nil && userHasMFAEnrolled(db, claims.UserID) {
				helpers.RespondWithError(w, 403, []string{"FORBIDDEN"})
				return
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func userHasMFAEnrolled(db *gorm.DB, userID uuid.UUID) bool {
	var count int64
	db.Model(&models.MFADevice{}).
		Where("user_id = ? AND is_verified = ?", userID, true).
		Count(&count)
	return count > 0
}

func isMFABypassPath(path, method string) bool {
	for _, rule := range configuration.MFABypassRules {
		if rule.Method != "*" && rule.Method != method {
			continue
		}

		if (rule.ExactPath != "" && rule.ExactPath == path) || (rule.Pattern != nil && rule.Pattern.MatchString(path)) {
			return true
		}
	}
	return false
}
