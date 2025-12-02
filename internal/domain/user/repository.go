package user

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("user not found")

type ListFilter struct {
	Limit  int
	Offset int
}

type Repository interface {
	GetById(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, filter ListFilter) ([]User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
}
