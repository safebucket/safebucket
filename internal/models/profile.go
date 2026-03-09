package models

type WorkerMode string

const (
	WorkerModeDisabled  WorkerMode = "disabled"  // Worker is disabled
	WorkerModeSingleton WorkerMode = "singleton" // Only one instance runs this worker
	WorkerModeAll       WorkerMode = "all"       // All instances run this worker
)

type Profile struct {
	Name       string
	HTTPServer bool
	Workers    WorkerConfig
}

type WorkerConfig struct {
	ObjectDeletion   WorkerMode
	BucketEvents     WorkerMode
	TrashCleanup     WorkerMode
	GarbageCollector WorkerMode
}

func (w WorkerConfig) AnyEnabled() bool {
	return w.ObjectDeletion != WorkerModeDisabled ||
		w.BucketEvents != WorkerModeDisabled ||
		w.TrashCleanup != WorkerModeDisabled ||
		w.GarbageCollector != WorkerModeDisabled
}

func (p Profile) NeedsEvents() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}
