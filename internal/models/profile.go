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

// AnySingleton returns true if any worker is configured as singleton.
func (w WorkerConfig) AnySingleton() bool {
	return w.ObjectDeletion == WorkerModeSingleton || w.BucketEvents == WorkerModeSingleton
}

// NeedsCache returns true if the profile requires cache configuration.
func (p Profile) NeedsCache() bool {
	return p.HTTPServer || p.Workers.AnySingleton()
}

// NeedsStorage returns true if the profile requires storage configuration.
func (p Profile) NeedsStorage() bool {
	return p.HTTPServer || p.Workers.ObjectDeletion != WorkerModeDisabled || p.Workers.BucketEvents != WorkerModeDisabled
}

// NeedsEvents returns true if the profile requires events configuration.
func (p Profile) NeedsEvents() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}

// NeedsNotifier returns true if the profile requires notifier configuration.
func (p Profile) NeedsNotifier() bool {
	return p.HTTPServer
}

// NeedsAuth returns true if the profile requires auth configuration.
func (p Profile) NeedsAuth() bool {
	return p.HTTPServer
}

// NeedsActivity returns true if the profile requires activity logger configuration.
func (p Profile) NeedsActivity() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}
