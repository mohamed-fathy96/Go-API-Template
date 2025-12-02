package user

import (
	"context"
	"encoding/json"
	"fmt"
	"kabsa/internal/cache"
	"kabsa/internal/db"
	dom "kabsa/internal/domain/user"
	"kabsa/internal/logging"
	"time"
)

type Service interface {
	List(ctx context.Context, input ListUsersInput) ([]UserDto, error)
	GetById(ctx context.Context, id int64) (*UserDto, error)
	Create(ctx context.Context, input CreateUserInput) (*UserDto, error)
	Update(ctx context.Context, input UpdateUserInput) (*UserDto, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo   dom.Repository
	cache  cache.UserCache
	tx     db.Transactor // optional, for multi-entity transactions
	events Events
	logger logging.Logger
}

func (s *service) List(ctx context.Context, input ListUsersInput) ([]UserDto, error) {
	filter := dom.ListFilter{
		Limit:  input.Limit,
		Offset: input.Offset,
	}

	users, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list users", "error", err)
		return nil, fmt.Errorf("list users: %w", err)
	}

	return toDTOs(users), nil
}

func (s *service) GetById(ctx context.Context, id int64) (*UserDto, error) {
	// 1) Check cache
	if data, err := s.cache.GetByID(ctx, id); err == nil && data != nil {
		var dto UserDto
		if err := json.Unmarshal(data, &dto); err == nil {
			return &dto, nil
		}
		// If unmarshal fails, log and fall through to DB
		s.logger.Error("failed to unmarshal user from cache", "error", err, "id", id)
	} else if err != nil {
		s.logger.Error("failed to get user from cache", "error", err, "id", id)
	}

	// 2) Fallback to DB
	u, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	dto := toDTO(u)

	// 3) Write to cache (best-effort)
	if data, err := json.Marshal(dto); err == nil {
		if err := s.cache.Set(ctx, dto.Id, data, defaultUserCacheTTL); err != nil {
			s.logger.Error("failed to set user cache", "error", err, "id", dto.Id)
		}
	} else {
		s.logger.Error("failed to marshal user for cache", "error", err, "id", dto.Id)
	}

	return dto, nil
}

func (s *service) Create(ctx context.Context, input CreateUserInput) (*UserDto, error) {
	u := &dom.User{
		Email: input.Email,
		Name:  input.Name,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		s.logger.Error("failed to create user", "error", err, "email", input.Email)
		return nil, fmt.Errorf("create user: %w", err)
	}

	dto := toDTO(u)

	// Cache
	if data, err := json.Marshal(dto); err == nil {
		if err := s.cache.Set(ctx, dto.Id, data, defaultUserCacheTTL); err != nil {
			s.logger.Error("failed to set user cache after create", "error", err, "id", dto.Id)
		}
	} else {
		s.logger.Error("failed to marshal user for cache after create", "error", err, "id", dto.Id)
	}

	// Events (unchanged)
	if err := s.events.UserCreated(ctx, dto); err != nil {
		s.logger.Error("failed to publish UserCreated event", "error", err, "id", dto.Id)
	}

	return dto, nil
}

func (s *service) Update(ctx context.Context, input UpdateUserInput) (*UserDto, error) {
	u, err := s.repo.GetById(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		u.Name = *input.Name
	}

	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.Error("failed to update user", "error", err, "id", input.ID)
		return nil, fmt.Errorf("update user: %w", err)
	}

	dto := toDTO(u)

	// Update cache
	if data, err := json.Marshal(dto); err == nil {
		if err := s.cache.Set(ctx, dto.Id, data, defaultUserCacheTTL); err != nil {
			s.logger.Error("failed to set user cache after update", "error", err, "id", dto.Id)
		}
	} else {
		s.logger.Error("failed to marshal user for cache after update", "error", err, "id", dto.Id)
	}

	if err := s.events.UserUpdated(ctx, dto); err != nil {
		s.logger.Error("failed to publish UserUpdated event", "error", err, "id", dto.Id)
	}

	return dto, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.cache.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete user cache after delete", "error", err, "id", id)
	}

	if err := s.events.UserDeleted(ctx, id); err != nil {
		s.logger.Error("failed to publish UserDeleted event", "error", err, "id", id)
	}

	return nil
}

const defaultUserCacheTTL = 5 * time.Minute

func NewService(
	repo dom.Repository,
	cache cache.UserCache,
	tx db.Transactor,
	events Events,
	logger logging.Logger,
) Service {
	return &service{
		repo:   repo,
		cache:  cache,
		tx:     tx,
		events: events,
		logger: logger.With("component", "user_service"),
	}
}
