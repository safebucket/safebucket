package configuration

type AuthRule struct {
	Path        string
	Method      string // empty means all methods
	RequireAuth bool   // true means require auth, false means exclude from auth
}

var AuthRulePrefixMatchPath = []AuthRule{
	{Path: "/api/v1/auth", Method: "*", RequireAuth: false},    // All /auth excluded
	{Path: "/api/v1/invites", Method: "*", RequireAuth: false}, // All /invites require auth
	{Path: "/api/v1/buckets", Method: "*", RequireAuth: true},  // All /buckets require auth
	{Path: "/api/v1/users", Method: "*", RequireAuth: true},    // All /users require auth
}

var AuthRuleExactMatchPath = map[string][]AuthRule{
	"/invites": {
		{Path: "/api/v1/invites", Method: "POST", RequireAuth: true}, // POST /invites requires auth
	},
}
