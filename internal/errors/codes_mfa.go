package apierrors

const (
	CodeMFANotEnabled                 = "MFA_NOT_ENABLED"
	CodeMFARequired                   = "MFA_REQUIRED"
	CodeMFASetupFailed                = "MFA_SETUP_FAILED"
	CodeMFASetupRestricted            = "MFA_SETUP_RESTRICTED"
	CodeMFAVerificationFailed         = "MFA_VERIFICATION_FAILED"
	CodeMFARateLimited                = "MFA_RATE_LIMITED"
	CodeInvalidMFACode                = "INVALID_MFA_CODE"
	CodeMFADeviceNotFound             = "MFA_DEVICE_NOT_FOUND"
	CodeMFADeviceAlreadyVerified      = "MFA_DEVICE_ALREADY_VERIFIED"
	CodeMFADeviceNameExists           = "MFA_DEVICE_NAME_EXISTS"
	CodeMaxMFADevicesReached          = "MAX_MFA_DEVICES_REACHED"
	CodeUnverifiedDeviceCannotDefault = "UNVERIFIED_DEVICE_CANNOT_BE_DEFAULT"
)
