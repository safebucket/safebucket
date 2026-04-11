package middlewares

import (
	"context"
	"github.com/safebucket/safebucket/internal/configuration"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func startSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return otel.Tracer(configuration.AppName).Start(ctx, "middleware."+name)
}

func spanReject(span trace.Span, msg string) {
	span.SetStatus(codes.Error, msg)
}
