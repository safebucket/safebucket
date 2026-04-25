package core

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tracing"

	"go.uber.org/zap"
)

func NewTracer(config models.TracingConfiguration) tracing.ITracer {
	if !config.Enabled {
		zap.L().Info("Tracing disabled")
		return nil
	}

	switch config.Type {
	case "tempo":
		if config.Tempo == nil {
			zap.L().Warn("Tempo tracing enabled but no tempo config provided, tracing disabled")
			return nil
		}
		if config.Tempo.Tags == nil {
			config.Tempo.Tags = make(map[string]string)
		}

		tracer, err := tracing.NewTempoTracer(*config.Tempo)
		if err != nil {
			zap.L().Error(
				"Failed to initialize Tempo tracer, continuing without tracing",
				zap.Error(err),
			)
			return nil
		}
		zap.L().Info(
			"Tracing enabled",
			zap.String("type", config.Type),
			zap.String("endpoint", config.Tempo.Endpoint),
			zap.String("service", config.Tempo.ServiceName),
		)
		return tracer
	default:
		zap.L().Warn("Unknown tracing type, tracing disabled", zap.String("type", config.Type))
		return nil
	}
}
