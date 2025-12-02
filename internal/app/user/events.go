package user

import "context"

type Events interface {
	UserCreated(ctx context.Context, u *UserDto) error
	UserUpdated(ctx context.Context, u *UserDto) error
	UserDeleted(ctx context.Context, id int64) error
}

// NoopEvents No-op implementation, useful for tests or if you don’t need events yet.
type NoopEvents struct{}

func (NoopEvents) UserCreated(ctx context.Context, u *UserDto) error { return nil }
func (NoopEvents) UserUpdated(ctx context.Context, u *UserDto) error { return nil }
func (NoopEvents) UserDeleted(ctx context.Context, id int64) error   { return nil }
