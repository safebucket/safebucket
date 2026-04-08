package tracing

import (
	"fmt"
	"runtime"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/grafana/pyroscope-go"
)

type PyroscopeTracer struct {
	profiler *pyroscope.Profiler
}

func NewPyroscopeTracer(cfg models.PyroscopeConfiguration) (*PyroscopeTracer, error) {
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	profiler, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ApplicationName,
		ServerAddress:   cfg.ServerAddress,
		UploadRate:      time.Duration(cfg.UploadRate) * time.Second,
		Tags:            cfg.Tags,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("starting pyroscope profiler: %w", err)
	}

	return &PyroscopeTracer{profiler: profiler}, nil
}

func (p *PyroscopeTracer) Stop() error {
	if p == nil || p.profiler == nil {
		return nil
	}
	return p.profiler.Stop()
}
