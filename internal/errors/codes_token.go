package apierrors

const (
	CodeTokenNotFound     = "TOKEN_NOT_FOUND"
	CodeTokenExpired      = "TOKEN_EXPIRED"
	CodeTokenRevoked      = "TOKEN_REVOKED"
	CodeTokenCreateFailed = "TOKEN_CREATE_FAILED"
	CodeTokenRevokeFailed = "TOKEN_REVOKE_FAILED"
	CodeTokenActionDenied = "TOKEN_ACTION_DENIED"
)

const (
	CodeServiceAccountNotFound      = "SERVICE_ACCOUNT_NOT_FOUND"
	CodeServiceAccountAdminRequired = "SERVICE_ACCOUNT_ADMIN_FLAG_REQUIRED"
)
