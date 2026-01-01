package configuration

import "api/internal/models"

// Profiles defines all available deployment profiles.
var Profiles = map[string]models.Profile{
	"full": {
		Name:       "full",
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion: true,
			BucketEvents:   true,
		},
	},
	"api": {
		Name:       "api",
		HTTPServer: true,
		Workers:    models.WorkerConfig{},
	},
	"worker": {
		Name:       "worker",
		HTTPServer: false,
		Workers: models.WorkerConfig{
			ObjectDeletion: true,
			BucketEvents:   true,
		},
	},
}

// GetProfile returns the profile by name. Returns the "full" profile if name is empty.
func GetProfile(name string) (models.Profile, bool) {
	if name == "" {
		return Profiles["full"], true
	}
	profile, ok := Profiles[name]
	return profile, ok
}
