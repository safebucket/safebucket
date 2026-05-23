package apierrors

const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeForbidden           = "FORBIDDEN"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeNotFound            = "NOT_FOUND"
	CodeInvalidUUID         = "INVALID_UUID"
	CodeInvalidRequest      = "INVALID_REQUEST"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	CodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
)

const (
	CodeCreateFailed = "CREATE_FAILED"
	CodeUpdateFailed = "UPDATE_FAILED"
	CodeDeleteFailed = "DELETE_FAILED"
	CodeFetchFailed  = "FETCH_FAILED"
)
