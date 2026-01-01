package configuration

import "api/internal/models"

// Profiles defines all available deployment profiles.
var Profiles = map[string]models.Profile{
	"default": {
		Name:       "default",
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeAll,
			BucketEvents:   models.WorkerModeAll,
		},
	},
	"api": {
		Name:       "api",
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeDisabled,
			BucketEvents:   models.WorkerModeDisabled,
		},
	},
	"worker": {
		Name:       "worker",
		HTTPServer: false,
		Workers: models.WorkerConfig{
			ObjectDeletion: models.WorkerModeSingleton,
			BucketEvents:   models.WorkerModeSingleton,
		},
	},
}

// GetProfile returns the profile by name. Returns the "full" profile if name is empty.
func GetProfile(name string) (models.Profile, bool) {
	profile, ok := Profiles[name]
	return profile, ok
}
