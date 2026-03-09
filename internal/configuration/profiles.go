package configuration

import (
	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
)

const (
	ProfileDefault = "default"
	ProfileAPI     = "api"
	ProfileWorker  = "worker"
)

var Profiles = map[string]models.Profile{
	ProfileDefault: {
		Name:       ProfileDefault,
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion:   models.WorkerModeAll,
			BucketEvents:     models.WorkerModeAll,
			TrashCleanup:     models.WorkerModeSingleton,
			GarbageCollector: models.WorkerModeSingleton,
		},
	},
	ProfileAPI: {
		Name:       ProfileAPI,
		HTTPServer: true,
		Workers: models.WorkerConfig{
			ObjectDeletion:   models.WorkerModeDisabled,
			BucketEvents:     models.WorkerModeDisabled,
			TrashCleanup:     models.WorkerModeDisabled,
			GarbageCollector: models.WorkerModeDisabled,
		},
	},
	ProfileWorker: {
		Name:       ProfileWorker,
		HTTPServer: false,
		Workers: models.WorkerConfig{
			ObjectDeletion:   models.WorkerModeSingleton,
			BucketEvents:     models.WorkerModeSingleton,
			TrashCleanup:     models.WorkerModeSingleton,
			GarbageCollector: models.WorkerModeSingleton,
		},
	},
}

func GetProfile(name string) models.Profile {
	if name == "" {
		name = ProfileDefault
	}

	profile, ok := Profiles[name]

	if !ok {
		zap.L().Fatal("Unknown profile",
			zap.String("profile", name),
			zap.Strings("available_profiles", []string{ProfileDefault, ProfileAPI, ProfileWorker}))
	}

	zap.L().Info("Loaded profile", zap.String("profile", profile.Name))

	return profile
}
