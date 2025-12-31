package models

// Profile defines which components run for a given deployment mode.
type Profile struct {
	Name          string
	HTTPServer    bool
	Migrations    bool
	CacheTicker   bool
	Workers       WorkerConfig
	ExitAfterInit bool
}

// WorkerConfig defines which workers are enabled.
type WorkerConfig struct {
	Notifications  bool
	ObjectDeletion bool
	BucketEvents   bool
}

// AnyEnabled returns true if any worker is enabled.
func (w WorkerConfig) AnyEnabled() bool {
	return w.Notifications || w.ObjectDeletion || w.BucketEvents
}

// NeedsCache returns true if the profile requires cache configuration.
func (p Profile) NeedsCache() bool {
	return p.HTTPServer || p.CacheTicker
}

// NeedsStorage returns true if the profile requires storage configuration.
func (p Profile) NeedsStorage() bool {
	return p.HTTPServer || p.Workers.ObjectDeletion || p.Workers.BucketEvents
}

// NeedsEvents returns true if the profile requires events configuration.
func (p Profile) NeedsEvents() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}

// NeedsNotifier returns true if the profile requires notifier configuration.
func (p Profile) NeedsNotifier() bool {
	return p.Workers.Notifications
}

// NeedsAuth returns true if the profile requires auth configuration.
func (p Profile) NeedsAuth() bool {
	return p.HTTPServer
}

// NeedsActivity returns true if the profile requires activity logger configuration.
func (p Profile) NeedsActivity() bool {
	return p.HTTPServer || p.Workers.AnyEnabled()
}
