package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"kabsa/internal/logging"
)

func useBaseMiddlewares(r chi.Router, logger logging.Logger, serviceName string) {
	// Request ID / Real IP / Recover
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Logging middleware (your own)
	r.Use(requestLoggingMiddleware(logger))

	// Optional: timeout middleware
	r.Use(middleware.Timeout(60 * time.Second))
}

// Example logging middleware – keep or adjust as you like.
func requestLoggingMiddleware(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
