package rbac

// const action variables
const (
	ActionAll    = "*" // action match any other actions
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionList   = "list"
)

// const resource variables
const (
	ResourceAll    = "*" // resource match any other resources
	ResourceBucket = "bucket"
	ResourceUser   = "user"
	ResourceFile   = "file"
	ResourceGuest  = "log"
)

// const role variables
const (
	RoleAdmin = "role::admin"
	RoleUser  = "role::user"
	RoleGuest = "role::guest"
)

// const group variables
const (
	GroupOwner       = "group::owner"
	GroupContributor = "group::contributor"
	GroupViewer      = "group::viewer"
)

// const scope variables
const (
	ScopeSystem  = "System"
	ScopeProject = "App"
)
