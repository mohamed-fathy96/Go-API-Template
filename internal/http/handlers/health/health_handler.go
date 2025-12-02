package health

import (
	"kabsa/internal/cache"
	"kabsa/internal/db"
	"kabsa/internal/http/responses"
	"net/http"
)

type Handler struct {
	db    *db.Client
	cache *cache.RedisClient
}

func NewHandler(dbClient *db.Client, redisClient *cache.RedisClient) *Handler {
	return &Handler{
		db:    dbClient,
		cache: redisClient,
	}
}

// Check godoc
//
//	@Summary		Health check
//	@Description	Returns readiness of dependencies
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	apidocs.HealthResponse
//	@Failure		500	{object}	apidocs.ErrorEnvelope
//	@Router			/health [get]
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	// For now, just return OK. Later you can add DB/Redis ping logic here.
	responses.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
