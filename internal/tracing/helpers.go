package tracing

import (
	"context"

	"github.com/safebucket/safebucket/internal/configuration"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer(configuration.AppName).Start(ctx, name)
}
