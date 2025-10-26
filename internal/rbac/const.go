package rbac

// Action represents an allowable operation in the RBAC system.
// Use String() to obtain its literal value.
type Action string

func (a Action) String() string {
	return string(a)
}

const (
	ActionCreate   = Action("create")
	ActionDelete   = Action("delete")
	ActionDownload = Action("download")
	ActionErase    = Action("erase")
	ActionRestore  = Action("restore")
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
	ResourceBucket = Resource("bucket")
	ResourceFile   = Resource("file")
)
