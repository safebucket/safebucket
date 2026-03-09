package rbac

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
	ActionUpdate   = Action("update")
)

type Resource string

func (r Resource) String() string {
	return string(r)
}

const (
	ResourceBucket    = Resource("bucket")
	ResourceFile      = Resource("file")
	ResourceFolder    = Resource("folder")
	ResourceUser      = Resource("user")
	ResourceMFADevice = Resource("mfa_device")
)
