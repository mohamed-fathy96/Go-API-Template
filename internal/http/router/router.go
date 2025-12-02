package router

import (
	"github.com/go-chi/chi/v5"
	"kabsa/internal/http/handlers/health"
	userhandler "kabsa/internal/http/handlers/user"
	"kabsa/internal/http/responses"
	"kabsa/internal/logging"
	"net/http"
)

func NewRouter(
	logger logging.Logger,
	serviceName string,
	healthHandler *health.Handler,
	userHandler *userhandler.Handler,
) chi.Router {
	r := chi.NewRouter()

	useBaseMiddlewares(r, logger, serviceName)

	r.Route("/api/v1", func(r chi.Router) {
		// Health
		r.Get("/health", healthHandler.Check)

		// User module
		r.Route("/users", func(r chi.Router) {
			r.Get("/", userHandler.List)
			r.Post("/", userHandler.Create)
			r.Get("/{id}", userHandler.GetByID)
			r.Put("/{id}", userHandler.Update)
			r.Delete("/{id}", userHandler.Delete)
		})
	})

	// Optionally: 404 handler
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		responses.WriteNotFound(w, r)
	})

	return r
}
