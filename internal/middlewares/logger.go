package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const LoggerKey contextKey = "logger"

func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Generate UUID request ID and create request-scoped logger
		requestID := uuid.New().String()
		logger := zap.L().With(zap.String("request_id", requestID))

		// Add request ID to response headers for debugging
		w.Header().Set("X-Request-ID", requestID)

		// Store logger in context
		ctx := context.WithValue(r.Context(), LoggerKey, logger)
		r = r.WithContext(ctx)

		t1 := time.Now()
		defer func() {
			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("proto", r.Proto),
				zap.Duration("lat", time.Since(t1)),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()))
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

func GetLogger(r *http.Request) *zap.Logger {
	if logger, ok := r.Context().Value(LoggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.L()
}
