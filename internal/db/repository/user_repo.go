package repository

import (
	"context"
	"fmt"
	"kabsa/ent"
	entuser "kabsa/ent/user"
	"kabsa/internal/db"
	dom "kabsa/internal/domain/user"
	"kabsa/internal/logging"
)

type UserRepository struct {
	client *db.Client
	logger logging.Logger
}

func NewUserRepository(client *db.Client, logger logging.Logger) dom.Repository {
	return &UserRepository{
		client: client,
		logger: logger.With("component", "user_repo"),
	}
}

func (r *UserRepository) GetById(ctx context.Context, id int64) (*dom.User, error) {
	u, err := r.client.Ent().User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, dom.ErrNotFound
		}
		return nil, fmt.Errorf("ent.User.Get: %w", err)
	}
	return toDomainUser(u), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*dom.User, error) {
	u, err := r.client.Ent().User.
		Query().
		Where(entuser.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, dom.ErrNotFound
		}
		return nil, fmt.Errorf("ent.User.Query: %w", err)
	}
	return toDomainUser(u), nil
}

func (r *UserRepository) List(ctx context.Context, filter dom.ListFilter) ([]dom.User, error) {
	q := r.client.Ent().User.Query()

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		q = q.Offset(filter.Offset)
	}

	users, err := q.Order(ent.Asc(entuser.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("ent.User.Query.All: %w", err)
	}

	return toDomainUsers(users), nil
}

func (r *UserRepository) Create(ctx context.Context, u *dom.User) error {
	created, err := r.client.Ent().User.
		Create().
		SetEmail(u.Email).
		SetName(u.Name).
		Save(ctx)
	if err != nil {
		// handle unique email violation, etc.
		return fmt.Errorf("ent.User.Create: %w", err)
	}

	// Update input entity with generated fields
	u.ID = created.ID
	u.CreatedAt = created.CreatedAt
	u.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *dom.User) error {
	_, err := r.client.Ent().User.
		UpdateOneID(u.ID).
		SetEmail(u.Email).
		SetName(u.Name).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom.ErrNotFound
		}
		return fmt.Errorf("ent.User.UpdateOneID.Save: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	err := r.client.Ent().User.
		DeleteOneID(id).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom.ErrNotFound
		}
		return fmt.Errorf("ent.User.DeleteOneID.Exec: %w", err)
	}
	return nil
}
