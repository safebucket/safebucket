package configuration

import "api/internal/models"

const (
	ProfileDefault = "default"
	ProfileAPI     = "api"
	ProfileWorker  = "worker"
	ProfileFull    = "full"
)

// Profiles defines all available deployment profiles.
var Profiles = map[string]models.Profile{
	ProfileDefault: {
		Name:       ProfileDefault,
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeAll,
			BucketEvents:   models.WorkerModeAll,
			TrashCleanup:   models.WorkerModeDisabled,
		},
	},
	ProfileFull: {
		Name:       ProfileFull,
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeAll,
			BucketEvents:   models.WorkerModeAll,
			TrashCleanup:   models.WorkerModeSingleton,
		},
	},
	ProfileAPI: {
		Name:       ProfileAPI,
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeDisabled,
			BucketEvents:   models.WorkerModeDisabled,
			TrashCleanup:   models.WorkerModeDisabled,
		},
	},
	ProfileWorker: {
		Name:       ProfileWorker,
		HTTPServer: false,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeSingleton,
			BucketEvents:   models.WorkerModeSingleton,
			TrashCleanup:   models.WorkerModeDisabled,
		},
	},
}

// GetProfile returns the profile by name. Returns the default profile if name is empty.
// Returns false if the profile name is not found.
func GetProfile(name string) (models.Profile, bool) {
	if name == "" {
		name = ProfileDefault
	}

	profile, ok := Profiles[name]
	return profile, ok
}
