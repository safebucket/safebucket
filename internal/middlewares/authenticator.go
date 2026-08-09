package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"go.uber.org/zap"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	dbsql "github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type AuthExcludedKey struct{}

func Authenticate(
	jwtSecret string, db *gorm.DB, c cache.ICache, refreshTokenExpiry int,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.StartSpan(r.Context(), "middleware.Authenticate")
			defer span.End()
			r = r.WithContext(ctx)

			excluded := isPathExcludedFromAuth(r.URL.Path, r.Method)
			ctx = context.WithValue(r.Context(), AuthExcludedKey{}, excluded)

			if excluded {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			var claims models.UserClaims
			var status int
			var code string

			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				claims, status, code = authenticateAPIToken(db, c, h)
			} else {
				claims, status, code = authenticateJWT(jwtSecret, c, r, refreshTokenExpiry)
			}

			if code != "" {
				helpers.RespondWithErrorCtx(ctx, w, status, []string{code})
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, models.UserClaimKey{}, claims)))
		}
		return http.HandlerFunc(fn)
	}
}

func authenticateAPIToken(
	db *gorm.DB,
	c cache.ICache,
	authHeader string,
) (models.UserClaims, int, string) {
	bearer, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || !helpers.HasAPITokenPrefix(bearer) {
		return models.UserClaims{}, http.StatusForbidden, apierrors.CodeForbidden
	}

	hash, err := helpers.ParseAPIToken(bearer)
	if err != nil {
		return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeTokenNotFound
	}

	var token models.APIToken
	err = db.Where("token_hash = ? AND deleted_at IS NULL", hash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeTokenNotFound
		}
		return models.UserClaims{}, http.StatusInternalServerError, apierrors.CodeInternalServerError
	}

	if token.RevokedAt != nil {
		return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeTokenRevoked
	}

	if token.ExpiresAt.Before(time.Now()) {
		return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeTokenExpired
	}

	user, err := dbsql.GetUserByID(db, token.UserID)
	if err != nil {
		return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeTokenNotFound
	}

	should, err := cache.ShouldUpdateTokenLastUsed(c, token.ID.String())
	if err != nil {
		zap.L().Warn("Failed to check api token last-used throttle", zap.Error(err))
	}

	if should {
		updErr := db.Model(&models.APIToken{}).
			Where("id = ?", token.ID.String()).
			Update("last_used_at", time.Now()).Error
		if updErr != nil {
			zap.L().Warn("Failed to update api token last_used_at", zap.Error(updErr))
		}
	}

	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Provider: string(user.ProviderType),
		MFA:      true,
		TokenID:  &token.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   configuration.AppName,
			Audience: jwt.ClaimStrings{configuration.AudienceAPIToken},
		},
	}
	return claims, 0, ""
}

func authenticateJWT(
	jwtSecret string,
	c cache.ICache,
	r *http.Request,
	refreshTokenExpiry int,
) (models.UserClaims, int, string) {
	var tokenStr string

	if mfaCookie, mfaErr := r.Cookie("safebucket_mfa_token"); mfaErr == nil {
		tokenStr = mfaCookie.Value
	} else if accessCookie, accessErr := r.Cookie("safebucket_access_token"); accessErr == nil {
		tokenStr = accessCookie.Value
	}

	userClaims, err := helpers.ParseToken(jwtSecret, tokenStr)
	if err != nil {
		return models.UserClaims{}, http.StatusForbidden, apierrors.CodeForbidden
	}

	if userClaims.Audience[0] == configuration.AudienceAccessToken {
		if userClaims.SID == "" {
			return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeSessionRevoked
		}

		maxAge := time.Duration(refreshTokenExpiry) * time.Minute
		active, sessionErr := cache.IsSessionActive(
			c, userClaims.UserID.String(), userClaims.SID, maxAge,
		)
		if sessionErr != nil || !active {
			return models.UserClaims{}, http.StatusUnauthorized, apierrors.CodeSessionRevoked
		}
	}

	return userClaims, 0, ""
}

func isPathExcludedFromAuth(path, method string) bool {
	if m, ok := configuration.AuthExcludedExactPaths[path]; ok {
		if m == "*" || m == method {
			return true
		}
	}

	for _, rule := range configuration.AuthExcludedPatterns {
		if rule.Pattern.MatchString(path) && (rule.Method == "*" || rule.Method == method) {
			return true
		}
	}

	return false
}
