package tracing

import (
	"context"
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type TempoTracer struct {
	provider *sdktrace.TracerProvider
}

func NewTempoTracer(cfg models.TempoConfiguration) (*TempoTracer, error) {
	exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpointURL(cfg.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	attrs := []attribute.KeyValue{semconv.ServiceNameKey.String(cfg.ServiceName)}
	for k, v := range cfg.Tags {
		attrs = append(attrs, attribute.String(k, v))
	}

	res, err := resource.New(context.Background(), resource.WithAttributes(attrs...))
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &TempoTracer{provider: provider}, nil
}

func (t *TempoTracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}
