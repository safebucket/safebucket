package configuration

import "api/internal/models"

// Profiles defines all available deployment profiles.
var Profiles = map[string]models.Profile{
	"full": {
		Name:        "full",
		HTTPServer:  true,
		Migrations:  true,
		CacheTicker: true,
		Workers: models.WorkerConfig{
			Notifications:  true,
			ObjectDeletion: true,
			BucketEvents:   true,
		},
	},
	"api": {
		Name:        "api",
		HTTPServer:  true,
		Migrations:  false,
		CacheTicker: true,
		Workers:     models.WorkerConfig{},
	},
	"worker": {
		Name:        "worker",
		HTTPServer:  false,
		Migrations:  false,
		CacheTicker: false,
		Workers: models.WorkerConfig{
			Notifications:  true,
			ObjectDeletion: true,
			BucketEvents:   true,
		},
	},
	"worker:notifications": {
		Name:        "worker:notifications",
		HTTPServer:  false,
		Migrations:  false,
		CacheTicker: false,
		Workers: models.WorkerConfig{
			Notifications: true,
		},
	},
	"worker:trash": {
		Name:        "worker:trash",
		HTTPServer:  false,
		Migrations:  false,
		CacheTicker: false,
		Workers: models.WorkerConfig{
			ObjectDeletion: true,
			BucketEvents:   true,
		},
	},
	"migrate": {
		Name:          "migrate",
		Migrations:    true,
		ExitAfterInit: true,
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
