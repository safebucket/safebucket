package core

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func NewTracer(config models.TracingConfiguration, component string) tracing.ITracer {
	if !config.Enabled {
		zap.L().Info("Tracing disabled")
		return nil
	}

	switch config.Type {
	case "pyroscope":
		if config.Pyroscope.Tags == nil {
			config.Pyroscope.Tags = make(map[string]string)
		}
		config.Pyroscope.Tags["component"] = component

		tracer, err := tracing.NewPyroscopeTracer(*config.Pyroscope)
		if err != nil {
			zap.L().Error(
				"Failed to initialize Pyroscope tracer, continuing without tracing",
				zap.Error(err),
			)
			return nil
		}
		zap.L().Info(
			"Tracing enabled",
			zap.String("type", config.Type),
			zap.String("server", config.Pyroscope.ServerAddress),
			zap.String("application", config.Pyroscope.ApplicationName),
		)
		return tracer
	default:
		zap.L().Warn("Unknown tracing type, tracing disabled", zap.String("type", config.Type))
		return nil
	}
}
