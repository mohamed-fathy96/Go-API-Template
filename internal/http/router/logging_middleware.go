package router

import (
	"github.com/go-chi/chi/v5/middleware"
	"kabsa/internal/logging"
	"net/http"
	"time"
)

func requestLogger(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Pull request ID from context if present
			reqID := middleware.GetReqID(r.Context())

			// You can also read traceID from context later if needed.

			next.ServeHTTP(ww, r)

			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", reqID,
				"remote_ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		}

		return http.HandlerFunc(fn)
	}
}
