package profiling

type IProfiler interface {
	Stop() error
}
