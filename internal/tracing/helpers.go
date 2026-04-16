package tracing

import (
	"context"
	"errors"

	"github.com/safebucket/safebucket/internal/configuration"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer(configuration.AppName).Start(ctx, name)
}

func RejectSpan(span trace.Span, msg string) {
	span.RecordError(errors.New(msg))
	span.SetStatus(codes.Error, msg)
}
