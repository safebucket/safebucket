package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			zap.L().Info("Request",
				zap.String("proto", r.Proto),
				zap.String("path", r.URL.Path),
				zap.Duration("lat", time.Since(t1)),
				zap.Int("status", ww.Status()),
				zap.Int("size", ww.BytesWritten()),
				zap.String("reqId", middleware.GetReqID(r.Context())))
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
