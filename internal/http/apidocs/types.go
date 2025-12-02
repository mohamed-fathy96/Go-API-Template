package apidocs

// HealthResponse is the shape of /health success.
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Redis   string `json:"redis" example:"ok"`
	DB      string `json:"db" example:"ok"`
	TraceID string `json:"traceId,omitempty"`
}

// UserResponse represents a user for responses.
type UserResponse struct {
	ID    int64  `json:"id" example:"1"`
	Name  string `json:"name" example:"Jane Doe"`
	Email string `json:"email" example:"jane@example.com"`
}

// UsersListResponse wraps a list.
type UsersListResponse struct {
	Data    []UserResponse `json:"data"`
	TraceID string         `json:"traceId,omitempty"`
}

// UserItemResponse wraps one item.
type UserItemResponse struct {
	Data    UserResponse `json:"data"`
	TraceID string       `json:"traceId,omitempty"`
}

// ErrorEnvelope matches your error writer.
type ErrorEnvelope struct {
	Code    string      `json:"code" example:"not_found"`
	Message string      `json:"message" example:"The requested resource was not found."`
	Details interface{} `json:"details,omitempty"`
	TraceID string      `json:"traceId,omitempty"`
}
