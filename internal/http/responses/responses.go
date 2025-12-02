package responses

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, ErrorResponse{Error: msg})
}

func WriteNotFound(w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotFound, "resource not found")
}

func WriteBadRequest(w http.ResponseWriter, msg string) {
	WriteError(w, http.StatusBadRequest, msg)
}
