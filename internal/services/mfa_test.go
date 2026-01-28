package services

import (
	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Inline Mocks ---

type MockCache struct {
}

func (m *MockCache) RegisterPlatform(_ string) error                   { return nil }
func (m *MockCache) DeleteInactivePlatform() error                     { return nil }
func (m *MockCache) StartIdentityTicker(_ string)                      {}
func (m *MockCache) GetRateLimit(_ string, _ int) (int, error)         { return 0, nil }
func (m *MockCache) IsTOTPCodeUsed(_ string, _ string) (bool, error)   { return false, nil }
func (m *MockCache) MarkTOTPCodeUsed(_ string, _ string) (bool, error) { return true, nil }
func (m *MockCache) GetMFAAttempts(_ string) (int, error)              { return 0, nil }
func (m *MockCache) IncrementMFAAttempts(_ string) error               { return nil }
func (m *MockCache) ResetMFAAttempts(_ string) error                   { return nil }
func (m *MockCache) Close() error                                      { return nil }

type MockNotifier struct {
}

func (m *MockNotifier) NotifyFromTemplate(_ string, _ string, _ string, _ interface{}) error {
	return nil
}

// --- Tests ---

func TestVerifyDevice_Security_PrivilegeEscalation(t *testing.T) {
	// Setup Token Helper (mocking JWT secret)
	jwtSecret := "test-secret"
	config := models.AuthConfig{
		JWTSecret:        jwtSecret,
		MFAEncryptionKey: "01234567890123456789012345678901", // 32 bytes
		WebURL:           "http://localhost:3000",
	}

	t.Run("should NOT return full access token when using Password Reset restricted token", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
		require.NoError(t, err)

		// Dependencies
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

		// 1. Create a Restricted Token (Password Reset Flow)
		// This simulates what the user has when entering the flow
		claims := models.UserClaims{
			UserID:      userID,
			Aud:         configuration.AudienceMFAReset,
			MFA:         false,
			ChallengeID: &challengeID,
		}

		// Mock DB interactions for VerifyDevice
		// 1. Transaction Start
		mock.ExpectBegin()

		// 2. Find User
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (id = $1 AND provider_type = $2) AND "users"."deleted_at" IS NULL`)).
			WithArgs(userID, models.LocalProviderType).
			WillReturnRows(userRow)

		// 3. Find Device (Locking)
		// We need a valid encrypted secret here to pass decryption
		// Using a dummy secret for the test
		encryptedSecret, _ := helpers.EncryptSecret("JBSWY3DPEHPK3PXP", []byte(config.MFAEncryptionKey))
		deviceRow := sqlmock.NewRows([]string{"id", "user_id", "encrypted_secret", "is_verified", "name"}).
			AddRow(deviceID.String(), userID.String(), encryptedSecret, false, "My Device")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WillReturnRows(deviceRow)

		// 4. Count existing default devices
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2 AND is_default = $3 AND id != $4`)).
			WithArgs(userID, true, true, deviceID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 5. Update Device
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "mfa_devices" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 6. Transaction Commit
		mock.ExpectCommit()

		// 7. Post-Transaction User Reload (DB Preload MFADevices)
		// This is called to generate the token
		// GORM generates: SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL AND "users"."id" = $2 ORDER BY "users"."id" LIMIT $3
		userRowReload := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, userID, 1).
			WillReturnRows(userRowReload)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices" WHERE "mfa_devices"."user_id" = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(deviceID))

		// Execute
		totpCode, _ := totp.GenerateCode("JBSWY3DPEHPK3PXP", time.Now())

		logger := zap.NewNop()

		response, err := service.VerifyDevice(
			logger, // legitimate logger to prevent panic
			claims,
			uuid.UUIDs{deviceID},
			models.MFADeviceVerifyBody{Code: totpCode},
		)
		require.NoError(t, err)

		// Assertions
		authResponse, ok := response.(models.AuthLoginResponse)
		require.True(t, ok)

		// Validate the returned Access Token
		// Use ParseToken for validation and check audience separately
		parsedClaims, err := helpers.ParseToken(jwtSecret, "Bearer "+authResponse.AccessToken, true)
		require.NoError(t, err, "Token should be parseable")

		// Verify this is a restricted token with the correct audience
		isRestrictedAudience := parsedClaims.Aud == configuration.AudienceMFALogin ||
			parsedClaims.Aud == configuration.AudienceMFAReset
		if !isRestrictedAudience {
			// If audience is full access, this is a vulnerability
			assert.NotEqual(t, configuration.AudienceAccessToken, parsedClaims.Aud,
				"VULNERABILITY CONFIRMED: Returned a valid Full Access Token instead of Restricted Token")
		} else {
			// If it's a restricted token, check Audience explicitly
			assert.Equal(t, configuration.AudienceMFAReset, parsedClaims.Aud,
				"Should preserve the Password Reset audience")

			// Also ensure MFA is marked as Verified
			assert.True(t, parsedClaims.MFA, "Restricted token should have MFA=true")
		}

		// Also assert NO Refresh Token is returned
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
				UserID:      userID,
				Aud:         tc.audience,
				MFA:         false,
				ChallengeID: &challengeID,
			}

			// Mock user lookup
			userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
				AddRow(userID, "test@example.com", models.LocalProviderType, "hash")
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
				WithArgs(userID, models.LocalProviderType, 1).
				WillReturnRows(userRow)

			// Mock device count (total)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
				WithArgs(userID).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			// Mock verified device count (CRITICAL: must be 0 for first device)
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
				WithArgs(userID, true).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			// Mock name uniqueness check
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
				WithArgs(userID, "My First Device", true).
				WillReturnRows(sqlmock.NewRows([]string{}))

			// Mock device creation
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
					Password: "", // No password required for restricted token first device
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
			UserID: userID,
			Aud:    configuration.AudienceMFALogin,
			MFA:    false,
		}

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock total device count
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock verified device count (1 = user already has a verified device)
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
			UserID: userID,
			Aud:    configuration.AudienceMFALogin,
			MFA:    false,
		}

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock total device count (3 unverified devices exist)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		// Mock verified device count (0 = no verified devices, just failed setup attempts)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Mock name uniqueness check
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "New Device", true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		// Mock device creation
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
			UserID: userID,
			Aud:    configuration.AudienceAccessToken,
			MFA:    true,
		}

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock device count
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
				Password: "", // Empty password
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
			UserID: userID,
			Aud:    configuration.AudienceAccessToken,
			MFA:    true,
		}

		// Create a valid password hash
		hashedPassword, _ := helpers.CreateHash("correct-password")

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, hashedPassword)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock device count
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
			UserID: userID,
			Aud:    configuration.AudienceAccessToken,
			MFA:    true,
		}

		hashedPassword, _ := helpers.CreateHash("correct-password")

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type", "hashed_password"}).
			AddRow(userID, "test@example.com", models.LocalProviderType, hashedPassword)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock device count
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock name uniqueness check
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "mfa_devices"`)).
			WithArgs(userID, "Second Device", true).
			WillReturnRows(sqlmock.NewRows([]string{}))

		// Mock device creation
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
			UserID: userID,
			Aud:    configuration.AudienceAccessToken,
			MFA:    false,
		}

		// Mock user lookup - OAuth user
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
			UserID: userID,
			Aud:    configuration.AudienceAccessToken,
			MFA:    true,
		}

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock device count (already at max)
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
			UserID: userID,
			Aud:    configuration.AudienceMFALogin,
			MFA:    false,
		}

		// Mock user lookup
		userRow := sqlmock.NewRows([]string{"id", "email", "provider_type"}).
			AddRow(userID, "test@example.com", models.LocalProviderType)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users"`)).
			WithArgs(userID, models.LocalProviderType, 1).
			WillReturnRows(userRow)

		// Mock device count
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices"`)).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Mock verified count
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(userID, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Mock name uniqueness check - name already exists
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
