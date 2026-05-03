package configuration

import (
	"net/http"
	"regexp"
)

// UUIDv4Pattern matches a valid UUID v4 format for use in path patterns.
const UUIDv4Pattern = `[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`

// AuthExcludedExactPaths lists exact paths excluded from authentication.
// Key is the exact path, value is the HTTP method ("*" for all methods).
var AuthExcludedExactPaths = map[string]string{
	"/api/v1/auth/login":          http.MethodPost,
	"/api/v1/auth/verify":         http.MethodPost,
	"/api/v1/auth/refresh":        http.MethodPost,
	"/api/v1/auth/reset-password": http.MethodPost,
}

// AuthPatternRule defines a regex pattern for path matching with dynamic segments.
type AuthPatternRule struct {
	Pattern *regexp.Regexp
	Method  string // "*" for all methods
}

// AuthExcludedPatterns lists regex patterns for paths with dynamic segments.
var AuthExcludedPatterns = []AuthPatternRule{
	{
		Pattern: regexp.MustCompile(
			`^/api/v1/auth/providers`,
		),
		Method: "*",
	},
	{
		Pattern: regexp.MustCompile(
			`^/api/v1/auth/reset-password/` + UUIDv4Pattern + `/validate$`,
		),
		Method: http.MethodPost,
	},
	{
		Pattern: regexp.MustCompile(
			`^/api/v1/shares/` + UUIDv4Pattern,
		),
		Method: "*",
	},
	{
		Pattern: regexp.MustCompile(
			`^/api/v1/invites/` + UUIDv4Pattern + `/challenges$`,
		),
		Method: http.MethodPost,
	},
	{
		Pattern: regexp.MustCompile(
			`^/api/v1/invites/` + UUIDv4Pattern + `/challenges/` + UUIDv4Pattern + `/validate$`,
		),
		Method: http.MethodPost,
	},
}

// AuthAudienceRule defines which token audiences are allowed for a specific route.
type AuthAudienceRule struct {
	ExactPath        string
	Pattern          *regexp.Regexp
	Method           string
	AllowedAudiences []string
}

// AuthAudienceRules defines the security policy for restricted token access.
// Routes not listed here will reject restricted tokens entirely.
var AuthAudienceRules = []AuthAudienceRule{
	{
		ExactPath:        "/api/v1/auth/mfa/verify",
		Method:           http.MethodPost,
		AllowedAudiences: []string{AudienceMFALogin, AudienceMFAReset},
	},
	{
		Pattern:          regexp.MustCompile(`^/api/v1/auth/reset-password/` + UUIDv4Pattern + `/complete$`),
		Method:           http.MethodPost,
		AllowedAudiences: []string{AudienceMFAReset},
	},
	{
		ExactPath:        "/api/v1/mfa/devices",
		Method:           "GET",
		AllowedAudiences: []string{AudienceAccessToken, AudienceMFALogin, AudienceMFAReset},
	},
	{
		ExactPath:        "/api/v1/mfa/devices",
		Method:           http.MethodPost,
		AllowedAudiences: []string{AudienceAccessToken, AudienceMFALogin, AudienceMFAReset},
	},
	{
		Pattern:          regexp.MustCompile(`^/api/v1/mfa/devices/` + UUIDv4Pattern + `/verify$`),
		Method:           http.MethodPost,
		AllowedAudiences: []string{AudienceAccessToken, AudienceMFALogin, AudienceMFAReset},
	},
}

// MFABypassRule defines paths that bypass MFA enforcement for full access tokens.
type MFABypassRule struct {
	ExactPath string
	Pattern   *regexp.Regexp
	Method    string
}

// MFABypassRules allows full access tokens without MFA to access these endpoints.
var MFABypassRules []MFABypassRule
