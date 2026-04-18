package tracing

import (
	"context"

	"github.com/safebucket/safebucket/internal/configuration"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

//nolint:spancheck // span is returned to the caller, which is responsible for End()
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer(configuration.AppName).Start(ctx, name)
}
