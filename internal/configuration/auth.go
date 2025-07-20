package configuration

type AuthRule struct {
	Path        string
	Method      string // empty means all methods
	RequireAuth bool   // true means require auth, false means exclude from auth
}

var AuthRulePrefixMatchPath = []AuthRule{
	{Path: "/auth", Method: "*", RequireAuth: false},    // All /auth excluded
	{Path: "/invites", Method: "*", RequireAuth: false}, // All /invites require auth
	{Path: "/buckets", Method: "*", RequireAuth: true},  // All /buckets require auth
	{Path: "/users", Method: "*", RequireAuth: true},    // All /users require auth
}

var AuthRuleExactMatchPath = map[string][]AuthRule{
	"/invites": {
		{Path: "/invites", Method: "POST", RequireAuth: true}, // POST /invites requires auth
	},
}
