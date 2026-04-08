package tracing

type ITracer interface {
	Stop() error
}
