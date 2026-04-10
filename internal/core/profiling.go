package core

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/profiling"

	"go.uber.org/zap"
)

func NewProfiler(config models.ProfilingConfiguration, component string) profiling.IProfiler {
	if !config.Enabled {
		zap.L().Info("Profiling disabled")
		return nil
	}

	switch config.Type {
	case "pyroscope":
		if config.Pyroscope.Tags == nil {
			config.Pyroscope.Tags = make(map[string]string)
		}
		config.Pyroscope.Tags["component"] = component

		tracer, err := profiling.NewPyroscopeProfiler(*config.Pyroscope)
		if err != nil {
			zap.L().Error(
				"Failed to initialize Pyroscope profiler, continuing without profiling",
				zap.Error(err),
			)
			return nil
		}
		zap.L().Info(
			"Profiling enabled",
			zap.String("type", config.Type),
			zap.String("server", config.Pyroscope.ServerAddress),
			zap.String("application", config.Pyroscope.ApplicationName),
		)
		return tracer
	default:
		zap.L().Warn("Unknown profiling type, profiling disabled", zap.String("type", config.Type))
		return nil
	}
}
