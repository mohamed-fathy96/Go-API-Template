package user

type CreateUserRequest struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name"  validate:"required,min=2,max=100"`
}

type UpdateUserRequest struct {
	Name *string `json:"name" validate:"omitempty,min=2,max=100"`
	// you can add more optional fields
}

type Response struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}
