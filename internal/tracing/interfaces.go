package tracing

import "context"

type ITracer interface {
	Shutdown(ctx context.Context) error
}
