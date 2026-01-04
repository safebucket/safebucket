package models

// WorkerMode defines how a worker should run.
type WorkerMode string

const (
	WorkerModeDisabled  WorkerMode = "disabled"  // Worker is disabled
	WorkerModeSingleton WorkerMode = "singleton" // Only one instance runs this worker
	WorkerModeAll       WorkerMode = "all"       // All instances run this worker
)

// Profile defines which components run for a given deployment mode.
type Profile struct {
	Name       string
	HTTPServer bool
	Workers    WorkerConfig
}

// WorkerConfig defines which workers are enabled and their mode.
type WorkerConfig struct {
	ObjectDeletion WorkerMode
	BucketEvents   WorkerMode
}

// AnyEnabled returns true if any worker is enabled.
func (w WorkerConfig) AnyEnabled() bool {
	return w.ObjectDeletion != WorkerModeDisabled || w.BucketEvents != WorkerModeDisabled
}

// NeedsEvents returns true if the profile requires events configuration.
func (p Profile) NeedsEvents() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}
