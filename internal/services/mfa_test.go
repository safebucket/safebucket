package services

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MockCache struct{}

func (m *MockCache) Get(_ string) (string, error)                            { return "", cache.ErrKeyNotFound }
func (m *MockCache) SetNX(_ string, _ string, _ time.Duration) (bool, error) { return true, nil }
func (m *MockCache) Del(_ string) error                                      { return nil }
func (m *MockCache) Incr(_ string) (int64, error)                            { return 1, nil }
func (m *MockCache) Expire(_ string, _ time.Duration) error                  { return nil }
func (m *MockCache) TTL(_ string) (time.Duration, error)                     { return 0, nil }
func (m *MockCache) ZAdd(_ string, _ float64, _ string) error                { return nil }
func (m *MockCache) ZRangeByScoreWithScores(_ string, _ string, _ string) ([]cache.ZScoreEntry, error) {
	return nil, nil
}
func (m *MockCache) ZScore(_ string, _ string) (float64, error)            { return 0, cache.ErrKeyNotFound }
func (m *MockCache) ZRemRangeByScore(_ string, _ string, _ string) error   { return nil }
func (m *MockCache) ScanKeys(_ string, _ int64, _ int64) ([]string, error) { return nil, nil }
func (m *MockCache) Close()                                                {}

type MockNotifier struct {
}

func (m *MockNotifier) NotifyFromTemplate(_ string, _ string, _ string, _ any) error {
	return nil
}

func TestVerifyDevice_Security_PrivilegeEscalation(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should NOT return full access token when using Password Reset restricted token", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		mockCache := &MockCache{}
		service := MFAService{
			DB:             gormDB,
			Cache:          mockCache,
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		deviceID := uuid.New()
		challengeID := uuid.New()

		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceMFAReset}},
			MFA:              false,
			ChallengeID:      &challengeID,
		}

		mock.ExpectBegin()

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		encryptedSecret, _ := helpers.EncryptSecret("JBSWY3DPEHPK3PXP", []byte(config.MFAEncryptionKey))
		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "encrypted_secret", "is_verified", "name"}).
			AddRow(deviceID.String(), userID.String(), encryptedSecret, false, "My Device")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WillReturnRows(deviceRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2 AND is_default = $3 AND id != $4`)).
			WithArgs(userID, true, true, deviceID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "mfa_devices" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		userRowReload := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, userID, 1).
			WillReturnRows(userRowReload)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices" WHERE "mfa_devices"."user_id" = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(deviceID))

		totpCode, _ := totp.GenerateCode("JBSWY3DPEHPK3PXP", time.Now())

		logger := zap.NewNop()

		response, err := service.VerifyDevice(
			false,
			logger,
			claims,
			uuid.UUIDs{deviceID},
			models.MFADeviceVerifyBody{Code: totpCode},
		)
		require.NoError(t, err)

		mfaToken := cookieValue(response, "safebucket_mfa_token")
		require.NotEmpty(t, mfaToken, "Password reset MFA verify should set the MFA cookie")

		parsedClaims, err := helpers.ParseToken(jwtSecret, mfaToken, false)
		require.NoError(t, err, "Token should be parseable")

		require.Equal(t, configuration.AudienceMFAReset, parsedClaims.Audience[0],
			"Should preserve the Password Reset audience")
		assert.True(t, parsedClaims.MFA, "Restricted token should have MFA=true")

		assert.Empty(t, cookieValue(response, "safebucket_refresh_token"),
			"Should not set the refresh cookie for password reset flow")
	})
}

func TestAddDevice_RestrictedToken_FirstDevice(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	testCases := []struct {
		name     string
		audience string
	}{
		{
			name:     "Login flow restricted token",
			audience: configuration.AudienceMFALogin,
		},
		{
			name:     "Password reset flow restricted token",
			audience: configuration.AudienceMFAReset,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func(db *sql.DB) { _ = db.Close() }(db)

			gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
			require.NoError(t, err)

			service := MFAService{
				DB:             gormDB,
				Cache:          &MockCache{},
				AuthConfig:     config,
				Notifier:       &MockNotifier{},
				ActivityLogger: &MockActivityLogger{},
			}

			userID := uuid.New()
			challengeID := uuid.New()

			claims := models.UserClaims{
				UserID:           userID,
				RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{tc.audience}},
				MFA:              false,
				ChallengeID:      &challengeID,
			}

			userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
				AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
				WithArgs(userID, 1).
				WillReturnRows(userRow)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
				WithArgs(userID).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
				WithArgs(userID, true).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
				WithArgs(userID, "My First Device", true).
				WillReturnRows(sqlmock.NewRows([]string{}))

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "mfa_devices"`)).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
			mock.ExpectCommit()

			logger := zap.NewNop()

			response, err := service.AddDevice(
				logger,
				claims,
				uuid.UUIDs{},
				models.MFADeviceSetupBody{
					Name:     "My First Device",
					Password: "",
				},
			)

			require.NoError(t, err)
			assert.NotEmpty(t, response.DeviceID)
			assert.NotEmpty(t, response.Secret)
			assert.NotEmpty(t, response.QRCodeURI)
		})
	}
}

func TestAddDevice_RestrictedToken_SecondDevice_ShouldFail(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should reject restricted token when user already has verified device", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceMFALogin}},
			MFA:              false,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Second Device",
				Password: "",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA_SETUP_RESTRICTED")
	})
}

func TestAddDevice_RestrictedToken_WithUnverifiedDevices(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should allow restricted token when user has unverified devices", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceMFALogin}},
			MFA:              false,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "New Device", true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "mfa_devices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		logger := zap.NewNop()

		response, err := service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "New Device",
				Password: "",
			},
		)

		require.NoError(t, err)
		assert.NotEmpty(t, response.DeviceID)
	})
}

func TestAddDevice_FullAccessToken_RequiresPassword(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should require password for full access token", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Second Device",
				Password: "",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "BAD_REQUEST")
	})

	t.Run("should reject invalid password", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}

		hashedPassword, _ := helpers.CreateHash("correct-password")

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, hashedPassword)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Second Device",
				Password: "wrong-password",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_PASSWORD")
	})

	t.Run("should succeed with valid password", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}

		hashedPassword, _ := helpers.CreateHash("correct-password")

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, hashedPassword)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "Second Device", true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "mfa_devices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		logger := zap.NewNop()

		response, err := service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Second Device",
				Password: "correct-password",
			},
		)

		require.NoError(t, err)
		assert.NotEmpty(t, response.DeviceID)
		assert.NotEmpty(t, response.Secret)
	})
}

func TestAddDevice_EdgeCases(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		TokenSecret:      jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should reject unknown user", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              false,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{}))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Device",
				Password: "password",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "USER_NOT_FOUND")
	})

	t.Run("should reject when max devices reached", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(configuration.MaxMFADevicesPerUser))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "Too Many",
				Password: "password",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MAX_MFA_DEVICES_REACHED")
	})

	t.Run("should reject duplicate device names", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceMFALogin}},
			MFA:              false,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		existingDeviceRow := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(uuid.New(), "My Device")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "My Device", true).
			WillReturnRows(existingDeviceRow)

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "My Device",
				Password: "",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MFA_DEVICE_NAME_EXISTS")
	})

	t.Run("should allow OIDC user without password", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              false,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "oidc@example.com", models.OIDCProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "OIDC Device", true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "mfa_devices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		logger := zap.NewNop()

		response, err := service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "OIDC Device",
				Password: "",
			},
		)

		require.NoError(t, err)
		assert.NotEmpty(t, response.DeviceID)
		assert.NotEmpty(t, response.Secret)
	})

	t.Run("should reject empty password for LDAP user", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}

		userID := uuid.New()
		claims := models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}

		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key"}).
			AddRow(userID, "jdoe@example.org", models.LDAPProviderType, "corp")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		logger := zap.NewNop()

		_, err = service.AddDevice(
			logger,
			claims,
			uuid.UUIDs{},
			models.MFADeviceSetupBody{
				Name:     "LDAP Device",
				Password: "",
			},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "BAD_REQUEST")
	})
}

func TestRemoveDevice_OIDCStepUp(t *testing.T) {
	config := models.AuthConfig{
		TokenSecret:      "test-secret",
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	newOIDCService := func(t *testing.T) (MFAService, sqlmock.Sqlmock, uuid.UUID) {
		t.Helper()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}
		return service, mock, uuid.New()
	}

	claimsFor := func(userID uuid.UUID) models.UserClaims {
		return models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}
	}

	expectUser := func(mock sqlmock.Sqlmock, userID uuid.UUID, providerType models.ProviderType, key string) {
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key"}).
			AddRow(userID, "user@example.com", providerType, key)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)
	}

	expectDeviceRead := func(mock sqlmock.Sqlmock, deviceID, userID uuid.UUID, verified bool) {
		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "is_default", "is_verified"}).
			AddRow(deviceID, userID, false, verified)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(deviceID, userID, 1).
			WillReturnRows(deviceRow)
	}

	t.Run("should reject removal without a code for OIDC user", func(t *testing.T) {
		service, mock, userID := newOIDCService(t)
		deviceID := uuid.New()

		expectUser(mock, userID, models.OIDCProviderType, "google")
		expectDeviceRead(mock, deviceID, userID, true)

		err := service.RemoveDevice(zap.NewNop(), claimsFor(userID), uuid.UUIDs{deviceID},
			models.MFADeviceRemoveBody{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "BAD_REQUEST")
	})

	t.Run("should reject removal with an invalid code for OIDC user", func(t *testing.T) {
		service, mock, userID := newOIDCService(t)
		deviceID := uuid.New()

		expectUser(mock, userID, models.OIDCProviderType, "google")
		expectDeviceRead(mock, deviceID, userID, true)

		encryptedSecret, err := helpers.EncryptSecret("JBSWY3DPEHPK3PXP", []byte(config.MFAEncryptionKey))
		require.NoError(t, err)
		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "encrypted_secret", "is_verified"}).
			AddRow(deviceID, userID, encryptedSecret, true)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, true).
			WillReturnRows(deviceRow)

		wrongCode, err := totp.GenerateCode("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Now())
		require.NoError(t, err)

		err = service.RemoveDevice(zap.NewNop(), claimsFor(userID), uuid.UUIDs{deviceID},
			models.MFADeviceRemoveBody{Code: wrongCode})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_MFA_CODE")
	})

	t.Run("should remove an unverified device without re-auth for OIDC user", func(t *testing.T) {
		service, mock, userID := newOIDCService(t)
		deviceID := uuid.New()

		expectUser(mock, userID, models.OIDCProviderType, "google")
		expectDeviceRead(mock, deviceID, userID, false)

		mock.ExpectBegin()
		expectDeviceRead(mock, deviceID, userID, false)
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "mfa_devices"`)).
			WithArgs(deviceID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := service.RemoveDevice(zap.NewNop(), claimsFor(userID), uuid.UUIDs{deviceID},
			models.MFADeviceRemoveBody{})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should remove an unverified device without a password for local user", func(t *testing.T) {
		service, mock, userID := newOIDCService(t)
		deviceID := uuid.New()

		expectUser(mock, userID, models.LocalProviderType, "local")
		expectDeviceRead(mock, deviceID, userID, false)

		mock.ExpectBegin()
		expectDeviceRead(mock, deviceID, userID, false)
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "mfa_devices"`)).
			WithArgs(deviceID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := service.RemoveDevice(zap.NewNop(), claimsFor(userID), uuid.UUIDs{deviceID},
			models.MFADeviceRemoveBody{})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should reject when the device is verified concurrently", func(t *testing.T) {
		service, mock, userID := newOIDCService(t)
		deviceID := uuid.New()

		expectUser(mock, userID, models.OIDCProviderType, "google")
		expectDeviceRead(mock, deviceID, userID, false)

		mock.ExpectBegin()
		expectDeviceRead(mock, deviceID, userID, true)
		mock.ExpectRollback()

		err := service.RemoveDevice(zap.NewNop(), claimsFor(userID), uuid.UUIDs{deviceID},
			models.MFADeviceRemoveBody{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "BAD_REQUEST")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAddDevice_OIDCStepUp(t *testing.T) {
	config := models.AuthConfig{
		TokenSecret:      "test-secret",
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	newService := func(t *testing.T) (MFAService, sqlmock.Sqlmock, uuid.UUID) {
		t.Helper()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		service := MFAService{
			DB:             gormDB,
			Cache:          &MockCache{},
			AuthConfig:     config,
			Notifier:       &MockNotifier{},
			ActivityLogger: &MockActivityLogger{},
		}
		return service, mock, uuid.New()
	}

	appClaims := func(userID uuid.UUID) models.UserClaims {
		return models.UserClaims{
			UserID:           userID,
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{configuration.AudienceAccessToken}},
			MFA:              true,
		}
	}

	expectUserAndCounts := func(mock sqlmock.Sqlmock, userID uuid.UUID) {
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "provider_key"}).
			AddRow(userID, "oidc@example.com", models.OIDCProviderType, "google")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, 1).
			WillReturnRows(userRow)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	}

	t.Run("should reject a second device without a code for OIDC user", func(t *testing.T) {
		service, mock, userID := newService(t)
		expectUserAndCounts(mock, userID)

		_, err := service.AddDevice(zap.NewNop(), appClaims(userID), uuid.UUIDs{},
			models.MFADeviceSetupBody{Name: "second"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "BAD_REQUEST")
	})

	t.Run("should reject a second device with an invalid code for OIDC user", func(t *testing.T) {
		service, mock, userID := newService(t)
		expectUserAndCounts(mock, userID)

		encryptedSecret, err := helpers.EncryptSecret("JBSWY3DPEHPK3PXP", []byte(config.MFAEncryptionKey))
		require.NoError(t, err)
		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "encrypted_secret", "is_verified"}).
			AddRow(uuid.New(), userID, encryptedSecret, true)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, true).
			WillReturnRows(deviceRow)

		wrongCode, err := totp.GenerateCode("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Now())
		require.NoError(t, err)

		_, err = service.AddDevice(zap.NewNop(), appClaims(userID), uuid.UUIDs{},
			models.MFADeviceSetupBody{Name: "second", Code: wrongCode})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "INVALID_MFA_CODE")
	})
}
