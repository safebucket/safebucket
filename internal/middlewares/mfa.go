package middlewares

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MFAValidate(
	db *gorm.DB,
	mfaRequired bool,
	providers configuration.Providers,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.MFAValidate")
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

			if claims.AudienceString() != configuration.AudienceAccessToken {
				next.ServeHTTP(w, r)
				return
			}

			isLocal := claims.Provider == string(models.LocalProviderType)
			providerRequiresMFA := false
			if p, found := providers[claims.Provider]; found {
				providerRequiresMFA = p.MFARequired
			}
			if !isLocal && !providerRequiresMFA {
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

			if (isLocal && mfaRequired) || providerRequiresMFA {
				helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
				return
			}

			if db != nil && userHasMFAEnrolled(db, claims.UserID) {
				helpers.RespondWithErrorCtx(r.Context(), w, 403, []string{apierrors.CodeForbidden})
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
