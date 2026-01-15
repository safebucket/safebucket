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
	"/api/v1/auth/mfa/verify": {
		{Path: "/api/v1/auth/mfa/verify", Method: "POST", RequireAuth: true},
	},
}

type MFABypassRule struct {
	PathPrefix string
	PathSuffix string
	Method     string
}

// MFABypassRules allows full access tokens without MFA to access these endpoints.
// Note: Users in MFA setup flow now use restricted tokens (auth:mfa audience) instead,
// which have their own allowed endpoints in the Authenticate middleware.
var MFABypassRules = []MFABypassRule{
	// Logout - always allowed
	{PathPrefix: "/api/v1/auth/logout", PathSuffix: "", Method: "*"},
}
