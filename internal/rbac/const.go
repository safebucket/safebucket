package rbac

// const action variables

type Action string

func (a Action) String() string {
	return string(a)
}

const (
	ActionAll      = Action("*") // action match any other actions
	ActionCreate   = Action("create")
	ActionDelete   = Action("delete")
	ActionErase    = Action("erase")
	ActionRead     = Action("read")
	ActionUpdate   = Action("update")
	ActionUpload   = Action("upload")
	ActionDownload = Action("download")
	ActionList     = Action("list")
)

type Resource string

func (r Resource) String() string {
	return string(r)
}

// const resource variables
const (
	ResourceAll    = Resource("*") // resource match any other resources
	ResourceBucket = Resource("bucket")
	ResourceUser   = Resource("user")
	ResourceFile   = Resource("file")
	ResourceLog    = Resource("log")
)

type Role string

func (r Role) String() string {
	return string(r)
}

// const role variables
const (
	RoleAdmin = Role("role::admin")
	RoleUser  = Role("role::user")
	RoleGuest = Role("role::guest")
)

type Group string

func (g Group) String() string {
	return string(g)
}

// const group variables
const (
	GroupOwner       = Group("group::owner")
	GroupContributor = Group("group::contributor")
	GroupViewer      = Group("group::viewer")
)

type Scope string

func (s Scope) String() string {
	return string(s)
}

// const scope variables
const (
	ScopeSystem  = Scope("System")
	ScopeProject = Scope("App")
)
