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

// Check is a simple health endpoint. You can later extend it to ping DB/Redis.
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	// For now, just return OK. Later you can add DB/Redis ping logic here.
	responses.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
