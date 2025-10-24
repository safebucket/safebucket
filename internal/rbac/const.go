package rbac

// Action represents an allowable operation in the RBAC system.
// Use String() to obtain its literal value.
type Action string

func (a Action) String() string {
	return string(a)
}

const (
	ActionAll      = Action("*") // action match any other actions
	ActionCreate   = Action("create")
	ActionDelete   = Action("delete")
	ActionDownload = Action("download")
	ActionErase    = Action("erase")
	ActionList     = Action("list")
	ActionRead     = Action("read")
	ActionRestore  = Action("restore")
	ActionUpdate   = Action("update")
	ActionUpload   = Action("upload")
	ActionGrant    = Action("grant")
	ActionPurge    = Action("purge")
)

// Resource represents an object type in the RBAC system.
type Resource string

func (r Resource) String() string {
	return string(r)
}

// Predefined Resource constants for common resources.
const (
	ResourceAll    = Resource("*") // resource match any other resources
	ResourceBucket = Resource("bucket")
	ResourceUser   = Resource("user")
	ResourceFile   = Resource("file")
	ResourceLog    = Resource("log")
	ResourceInvite = Resource("invite")
)

type Role string

func (r Role) String() string {
	return string(r)
}

// Role represents a high-level user role, such as admin or guest.
const (
	RoleAdmin = Role("role::admin")
	RoleUser  = Role("role::user")
	RoleGuest = Role("role::guest")
)

type Group string

func (g Group) String() string {
	return string(g)
}

// Predefined Group constants for access groups.
const (
	GroupOwner       = Group("group::owner")
	GroupContributor = Group("group::contributor")
	GroupViewer      = Group("group::viewer")
)

// Scope represents a permission scope, such as System or App.
type Scope string

func (s Scope) String() string {
	return string(s)
}

// const scope variables
const (
	ScopeSystem  = Scope("System")
	ScopeProject = Scope("App")
)
