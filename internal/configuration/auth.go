package configuration

import (
	"net/http"
	"regexp"
)

const UUIDv4Pattern = `[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`

var AuthExcludedExactPaths = map[string]string{
	"/api/v1/auth/login":          http.MethodPost,
	"/api/v1/auth/verify":         http.MethodPost,
	"/api/v1/auth/refresh":        http.MethodPost,
	"/api/v1/auth/reset-password": http.MethodPost,
}

type AuthPatternRule struct {
	Pattern *regexp.Regexp
	Method  string
}

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

type AuthAudienceRule struct {
	ExactPath        string
	Pattern          *regexp.Regexp
	Method           string
	AllowedAudiences []string
}

// AuthAudienceRules defines the audience policy; routes not listed here reject restricted tokens.
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

type MFABypassRule struct {
	ExactPath string
	Pattern   *regexp.Regexp
	Method    string
}

var MFABypassRules []MFABypassRule
