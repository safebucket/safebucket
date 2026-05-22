package services

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPasswordResetTestService(t *testing.T) (AuthPasswordResetService, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)

	svc := AuthPasswordResetService{
		DB:             gormDB,
		Cache:          &MockCache{},
		AuthConfig:     models.AuthConfig{TokenSecret: "test-secret"},
		ActivityLogger: &MockActivityLogger{},
	}
	cleanup := func() { _ = db.Close() }
	return svc, mock, cleanup
}

func requireAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	require.Error(t, err)
	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr), "expected APIError, got %T: %v", err, err)
	assert.Equal(t, status, apiErr.Status)
	assert.Equal(t, code, apiErr.Code)
}

func TestRequestPasswordReset(t *testing.T) {
	t.Run("returns success without creating challenge for non-matching email", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs("nobody@example.com", models.LocalProviderType, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		result, err := svc.RequestPasswordReset(
			zap.NewNop(),
			models.UserClaims{},
			uuid.UUIDs{},
			models.PasswordResetRequestBody{Email: "nobody@example.com"},
		)

		require.NoError(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompletePasswordReset(t *testing.T) {
	t.Run("rejects when JWT challenge_id differs from URL", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		urlChallengeID := uuid.New()
		jwtChallengeID := uuid.New()

		_, err := svc.CompletePasswordReset(false,
			zap.NewNop(),
			models.UserClaims{UserID: uuid.New(), ChallengeID: &jwtChallengeID},
			uuid.UUIDs{urlChallengeID},
			models.PasswordResetCompleteBody{NewPassword: "irrelevant"},
		)

		requireAPIError(t, err, 400, "INVALID_REQUEST")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects when JWT carries no challenge_id", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		_, err := svc.CompletePasswordReset(false,
			zap.NewNop(),
			models.UserClaims{UserID: uuid.New(), ChallengeID: nil},
			uuid.UUIDs{uuid.New()},
			models.PasswordResetCompleteBody{NewPassword: "irrelevant"},
		)

		requireAPIError(t, err, 400, "INVALID_REQUEST")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects when challenge is not bound to the JWT user", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		challengeID := uuid.New()
		jwtUserID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "challenges"`).
			WithArgs(challengeID, models.ChallengeTypePasswordReset, jwtUserID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		_, err := svc.CompletePasswordReset(false,
			zap.NewNop(),
			models.UserClaims{UserID: jwtUserID, ChallengeID: &challengeID},
			uuid.UUIDs{challengeID},
			models.PasswordResetCompleteBody{NewPassword: "irrelevant"},
		)

		requireAPIError(t, err, 400, "INVALID_REQUEST")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects expired challenge and deletes it", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		challengeID := uuid.New()
		userID := uuid.New()
		past := time.Now().Add(-time.Minute)

		challengeRow := sqlmock.NewRows(
			[]string{"id", "type", "hashed_secret", "attempts_left", "expires_at", "user_id"},
		).AddRow(challengeID, models.ChallengeTypePasswordReset, "hash", 3, past, userID)
		mock.ExpectQuery(`SELECT \* FROM "challenges"`).
			WithArgs(challengeID, models.ChallengeTypePasswordReset, userID, 1).
			WillReturnRows(challengeRow)

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "expired@example.com", models.LocalProviderType)
		mock.ExpectQuery(`SELECT \* FROM "users"`).
			WillReturnRows(userRow)

		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM "challenges"`).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		_, err := svc.CompletePasswordReset(false,
			zap.NewNop(),
			models.UserClaims{UserID: userID, ChallengeID: &challengeID},
			uuid.UUIDs{challengeID},
			models.PasswordResetCompleteBody{NewPassword: "irrelevant"},
		)

		requireAPIError(t, err, 400, "INVALID_REQUEST")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("blocks MFA bypass when MFA-enabled user has unverified token", func(t *testing.T) {
		svc, mock, cleanup := newPasswordResetTestService(t)
		defer cleanup()

		challengeID := uuid.New()
		userID := uuid.New()
		future := time.Now().Add(5 * time.Minute)

		challengeRow := sqlmock.NewRows(
			[]string{"id", "type", "hashed_secret", "attempts_left", "expires_at", "user_id"},
		).AddRow(challengeID, models.ChallengeTypePasswordReset, "hash", 3, future, userID)
		mock.ExpectQuery(`SELECT \* FROM "challenges"`).
			WithArgs(challengeID, models.ChallengeTypePasswordReset, userID, 1).
			WillReturnRows(challengeRow)

		challengeUserRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "mfa-user@example.com", models.LocalProviderType)
		mock.ExpectQuery(`SELECT \* FROM "users"`).
			WillReturnRows(challengeUserRow)

		mfaUserRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "mfa-user@example.com", models.LocalProviderType)
		mock.ExpectQuery(`SELECT \* FROM "users" WHERE id = `).
			WithArgs(userID, 1).
			WillReturnRows(mfaUserRow)

		mfaDeviceRows := sqlmock.NewRows([]string{"id", "user_id", "is_verified", "is_default"}).
			AddRow(uuid.New(), userID, true, true)
		mock.ExpectQuery(`SELECT \* FROM "mfa_devices"`).
			WillReturnRows(mfaDeviceRows)

		_, err := svc.CompletePasswordReset(false,
			zap.NewNop(),
			models.UserClaims{UserID: userID, ChallengeID: &challengeID, MFA: false},
			uuid.UUIDs{challengeID},
			models.PasswordResetCompleteBody{NewPassword: "irrelevant"},
		)

		requireAPIError(t, err, 403, "MFA_REQUIRED")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestValidatePasswordReset_ExpiredChallenge(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func(db *sql.DB) { _ = db.Close() }(db)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)

	svc := AuthPasswordResetService{
		DB:             gormDB,
		Cache:          &MockCache{},
		AuthConfig:     models.AuthConfig{TokenSecret: "test-secret"},
		ActivityLogger: &MockActivityLogger{},
	}

	challengeID := uuid.New()
	userID := uuid.New()
	past := time.Now().Add(-time.Minute)

	mock.ExpectBegin()

	challengeRow := sqlmock.NewRows(
		[]string{"id", "type", "hashed_secret", "attempts_left", "expires_at", "user_id"},
	).AddRow(challengeID, models.ChallengeTypePasswordReset, "hash", 3, past, userID)
	mock.ExpectQuery(`SELECT \* FROM "challenges"`).
		WithArgs(challengeID, models.ChallengeTypePasswordReset, 1).
		WillReturnRows(challengeRow)

	userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
		AddRow(userID, "expired@example.com", models.LocalProviderType)
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(userRow)

	mock.ExpectExec(`DELETE FROM "challenges"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectRollback()

	_, err = svc.ValidatePasswordReset(false,
		zap.NewNop(),
		models.UserClaims{},
		uuid.UUIDs{challengeID},
		models.PasswordResetValidateBody{Code: "ABCDEF"},
	)

	requireAPIError(t, err, 400, "INVALID_REQUEST")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAudienceConstants_Unique(t *testing.T) {
	audiences := []string{
		configuration.AudienceAccessToken,
		configuration.AudienceRefreshToken,
		configuration.AudienceMFALogin,
		configuration.AudienceMFAReset,
	}

	seen := make(map[string]struct{}, len(audiences))
	for _, aud := range audiences {
		require.NotEmpty(t, aud)
		_, dup := seen[aud]
		assert.False(t, dup, "audience %q must be unique", aud)
		seen[aud] = struct{}{}
	}
}
