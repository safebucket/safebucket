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
func (m *MockCache) ZScore(_ string, _ string) (float64, error)          { return 0, cache.ErrKeyNotFound }
func (m *MockCache) ZRemRangeByScore(_ string, _ string, _ string) error { return nil }
func (m *MockCache) Close()                                              {}

type MockNotifier struct {
}

func (m *MockNotifier) NotifyFromTemplate(_ string, _ string, _ string, _ interface{}) error {
	return nil
}

func TestVerifyDevice_Security_PrivilegeEscalation(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
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
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (id = $1 AND provider_type = $2) AND "users"."deleted_at" IS NULL`)).
			WithArgs(userID, models.LocalProviderType).
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
			logger,
			claims,
			uuid.UUIDs{deviceID},
			models.MFADeviceVerifyBody{Code: totpCode},
		)
		require.NoError(t, err)

		authResponse, ok := response.(models.AuthLoginResponse)
		require.True(t, ok)

		parsedClaims, err := helpers.ParseToken(jwtSecret, "Bearer "+authResponse.AccessToken, true)
		require.NoError(t, err, "Token should be parseable")

		isRestrictedAudience := parsedClaims.Audience[0] == configuration.AudienceMFALogin ||
			parsedClaims.Audience[0] == configuration.AudienceMFAReset
		if !isRestrictedAudience {
			assert.NotEqual(t, configuration.AudienceAccessToken, parsedClaims.Audience[0],
				"VULNERABILITY CONFIRMED: Returned a valid Full Access Token instead of Restricted Token")
		} else {
			assert.Equal(t, configuration.AudienceMFAReset, parsedClaims.Audience[0],
				"Should preserve the Password Reset audience")

			assert.True(t, parsedClaims.MFA, "Restricted token should have MFA=true")
		}

		assert.Empty(t, authResponse.RefreshToken, "Should not return Refresh Token for password reset flow")
	})
}

func TestAddDevice_RestrictedToken_FirstDevice(t *testing.T) {
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
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
				WithArgs(userID, models.LocalProviderType, 1).
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
		JWTSecret:        jwtSecret,
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
			WithArgs(userID, models.LocalProviderType, 1).
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
		JWTSecret:        jwtSecret,
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
			WithArgs(userID, models.LocalProviderType, 1).
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
		JWTSecret:        jwtSecret,
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
			WithArgs(userID, models.LocalProviderType, 1).
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
			WithArgs(userID, models.LocalProviderType, 1).
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
			WithArgs(userID, models.LocalProviderType, 1).
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
		JWTSecret:        jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901",
		WebURL:           "http://localhost:3000",
	}

	t.Run("should reject OAuth users", func(t *testing.T) {
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
			WithArgs(userID, models.LocalProviderType, 1).
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
			WithArgs(userID, models.LocalProviderType, 1).
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
			WithArgs(userID, models.LocalProviderType, 1).
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
}
