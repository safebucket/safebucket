package configuration

type AuthRule struct {
	Path        string
	Method      string // empty means all methods
	RequireAuth bool   // true means require auth, false means exclude from auth
}

var AuthRulePrefixMatchPath = []AuthRule{
	{Path: "/api/v1/auth", Method: "*", RequireAuth: false},
	{Path: "/api/v1/invites", Method: "*", RequireAuth: false},
	{Path: "/api/v1/buckets", Method: "*", RequireAuth: true},
	{Path: "/api/v1/users", Method: "*", RequireAuth: true},
	{Path: "/api/v1/settings", Method: "*", RequireAuth: true},
}

var AuthRuleExactMatchPath = map[string][]AuthRule{
	"/invites": {
		{Path: "/api/v1/invites", Method: "POST", RequireAuth: true},
	},
}

type MFABypassRule struct {
	PathPrefix string
	PathSuffix string
	Method     string
}

var MFABypassRules = []MFABypassRule{
	// Legacy single-device MFA setup endpoints
	{PathPrefix: "/api/v1/users/", PathSuffix: "/mfa/setup", Method: "POST"},
	{PathPrefix: "/api/v1/users/", PathSuffix: "/mfa/verify", Method: "POST"},

	// Multi-device MFA endpoints (for users setting up MFA)
	{PathPrefix: "/api/v1/users/", PathSuffix: "/mfa/devices", Method: "GET"},
	{PathPrefix: "/api/v1/users/", PathSuffix: "/mfa/devices", Method: "POST"},
	{PathPrefix: "/api/v1/users/", PathSuffix: "/verify", Method: "POST"}, // Matches /mfa/devices/{id}/verify

	// Logout
	{PathPrefix: "/api/v1/auth/logout", PathSuffix: "", Method: "*"},
}
