package services

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MockActivityLogger struct{}

func (m *MockActivityLogger) Send(_ models.Activity) error { return nil }
func (m *MockActivityLogger) Search(_ map[string][]string) ([]map[string]any, error) {
	return nil, nil
}
func (m *MockActivityLogger) CountByDay(_ map[string][]string, _ int) ([]models.TimeSeriesPoint, error) {
	return nil, nil
}
func (m *MockActivityLogger) Close() error { return nil }

var _ activity.IActivityLogger = (*MockActivityLogger)(nil)

func TestLogin_UserHasMFA_ConfigMFADisabled_RequiresMFA(t *testing.T) {
	jwtSecret := "test-secret-key-for-jwt-signing"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		MFARequired:      false,
		WebURL:           "http://localhost:3000",
	}

	t.Run("should require MFA when user has verified devices even if config MFA is disabled", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := AuthService{
			DB:         gormDB,
			Cache:      &MockCache{},
			AuthConfig: config,
			Providers: configuration.Providers{
				"local": {
					Name:    "Local",
					Type:    models.LocalProviderType,
					Domains: []string{}, // Allow all domains
				},
			},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		deviceID := uuid.New()

		hashedPassword, err := helpers.CreateHash("correct-password")
		require.NoError(t, err)

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key", "hashed_password", "role"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "local", hashedPassword, models.RoleUser)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND provider_type = $2 AND provider_key = $3) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $4`)).
			WithArgs("test@example.com", models.LocalProviderType, "local", 1).
			WillReturnRows(userRow)

		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "name", "is_verified", "is_default"}).
			AddRow(deviceID, userID, "My Authenticator", true, true)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices" WHERE "mfa_devices"."user_id" = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(deviceRow)

		logger := zap.NewNop()

		response, err := service.Login(
			logger,
			models.UserClaims{},
			uuid.UUIDs{},
			models.AuthLoginBody{
				Email:    "test@example.com",
				Password: "correct-password",
			},
		)

		require.NoError(t, err)

		assert.True(t, response.MFARequired,
			"MFARequired should be true when user has verified MFA devices")

		assert.NotEmpty(t, response.AccessToken, "Should return an access token")

		parsedClaims, err := helpers.ParseToken(jwtSecret, "Bearer "+response.AccessToken, true)
		require.NoError(t, err, "Token should be parseable")

		assert.Equal(t, configuration.AudienceMFALogin, parsedClaims.Audience[0],
			"Token should have AudienceMFALogin audience for MFA verification flow")

		assert.False(t, parsedClaims.MFA,
			"MFA claim should be false in restricted token before verification")

		assert.Empty(t, response.RefreshToken,
			"RefreshToken should be empty when MFA verification is required")

		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}

func TestLogin_UserNoMFA_ConfigMFADisabled_NoMFARequired(t *testing.T) {
	jwtSecret := "test-secret-key-for-jwt-signing"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		MFARequired:      false,
		WebURL:           "http://localhost:3000",
	}

	t.Run("should return full tokens when user has no MFA and config MFA is disabled", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := AuthService{
			DB:         gormDB,
			Cache:      &MockCache{},
			AuthConfig: config,
			Providers: configuration.Providers{
				"local": {
					Name:    "Local",
					Type:    models.LocalProviderType,
					Domains: []string{},
				},
			},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()

		hashedPassword, err := helpers.CreateHash("correct-password")
		require.NoError(t, err)

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key", "hashed_password", "role"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "local", hashedPassword, models.RoleUser)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND provider_type = $2 AND provider_key = $3) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $4`)).
			WithArgs("test@example.com", models.LocalProviderType, "local", 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices" WHERE "mfa_devices"."user_id" = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		logger := zap.NewNop()

		response, err := service.Login(
			logger,
			models.UserClaims{},
			uuid.UUIDs{},
			models.AuthLoginBody{
				Email:    "test@example.com",
				Password: "correct-password",
			},
		)

		require.NoError(t, err)

		assert.False(t, response.MFARequired,
			"MFARequired should be false when user has no MFA and config MFA is disabled")

		assert.NotEmpty(t, response.AccessToken)
		parsedClaims, err := helpers.ParseToken(jwtSecret, "Bearer "+response.AccessToken, true)
		require.NoError(t, err)
		assert.Equal(t, configuration.AudienceAccessToken, parsedClaims.Audience[0],
			"Should return full access token")

		assert.NotEmpty(t, response.RefreshToken,
			"Should return refresh token for full access")

		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}

func TestLogin_UserNoMFA_ConfigMFAEnabled_RequiresMFA(t *testing.T) {
	jwtSecret := "test-secret-key-for-jwt-signing"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		MFARequired:      true,
		WebURL:           "http://localhost:3000",
	}

	t.Run("should require MFA when config MFA is enabled even if user has no devices", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := AuthService{
			DB:         gormDB,
			Cache:      &MockCache{},
			AuthConfig: config,
			Providers: configuration.Providers{
				"local": {
					Name:    "Local",
					Type:    models.LocalProviderType,
					Domains: []string{},
				},
			},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()

		hashedPassword, err := helpers.CreateHash("correct-password")
		require.NoError(t, err)

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key", "hashed_password", "role"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "local", hashedPassword, models.RoleUser)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND provider_type = $2 AND provider_key = $3) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $4`)).
			WithArgs("test@example.com", models.LocalProviderType, "local", 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices" WHERE "mfa_devices"."user_id" = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		logger := zap.NewNop()

		response, err := service.Login(
			logger,
			models.UserClaims{},
			uuid.UUIDs{},
			models.AuthLoginBody{
				Email:    "test@example.com",
				Password: "correct-password",
			},
		)

		require.NoError(t, err)

		assert.True(t, response.MFARequired,
			"MFARequired should be true when config MFA is enabled")

		parsedClaims, err := helpers.ParseToken(jwtSecret, "Bearer "+response.AccessToken, true)
		require.NoError(t, err)
		assert.Equal(t, configuration.AudienceMFALogin, parsedClaims.Audience[0],
			"Should return restricted token for MFA setup")

		assert.Empty(t, response.RefreshToken)

		err = mock.ExpectationsWereMet()
		require.NoError(t, err)
	})
}
