package cache

type ICache interface {
	RegisterPlatform(id string) error
	DeleteInactivePlatform() error
	StartIdentityTicker(id string)

	GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error)

	// IsTOTPCodeUsed checks if a TOTP code has been used for a specific device.
	// The deviceID parameter identifies the MFA device (or user ID for legacy single-device users).
	IsTOTPCodeUsed(deviceID string, code string) (bool, error)
	// MarkTOTPCodeUsed marks a TOTP code as used for a specific device.
	// Uses configuration.TOTPCodeTTL constant for TTL.
	MarkTOTPCodeUsed(deviceID string, code string) error

	// GetMFAAttempts returns the current number of failed MFA attempts for a user.
	GetMFAAttempts(userID string) (int, error)
	// IncrementMFAAttempts increments failed MFA attempts and sets lockout TTL.
	// Uses configuration.MFALockoutSeconds constant for lockout duration.
	IncrementMFAAttempts(userID string) error
	// ResetMFAAttempts clears the failed attempts counter (called on successful verification).
	ResetMFAAttempts(userID string) error

	Close() error
}
