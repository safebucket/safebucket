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

var ValidResources []string

func defineResource(name string) Resource {
	ValidResources = append(ValidResources, name)
	return Resource(name)
}

var (
	ResourceBucket    = defineResource("bucket")
	ResourceFile      = defineResource("file")
	ResourceFolder    = defineResource("folder")
	ResourceUser      = defineResource("user")
	ResourceMFADevice = defineResource("mfa_device")
	ResourceShare     = defineResource("share")
)
