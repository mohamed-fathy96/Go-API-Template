package kafka

import (
	"context"
	"fmt"
	appuser "kabsa/internal/app/user"
	"kabsa/internal/config"
	"kabsa/internal/logging"
)

const (
	UserCreatedType = "UserCreated"
	UserUpdatedType = "UserUpdated"
	UserDeletedType = "UserDeleted"
)

type userEvents struct {
	bus         Bus
	topicPrefix string
	logger      logging.Logger
}

func NewUserEvents(bus Bus, cfg config.KafkaConfig, logger logging.Logger) appuser.Events {
	return &userEvents{
		bus:         bus,
		topicPrefix: cfg.TopicPrefix,
		logger:      logger.With("component", "user_events"),
	}
}

func (e *userEvents) topic() string {
	return e.topicPrefix + "users"
}

func (e *userEvents) UserCreated(ctx context.Context, u *appuser.UserDto) error {
	if err := e.bus.Publish(ctx, e.topic(), UserCreatedType, u); err != nil {
		return fmt.Errorf("publish UserCreated: %w", err)
	}
	return nil
}

func (e *userEvents) UserUpdated(ctx context.Context, u *appuser.UserDto) error {
	if err := e.bus.Publish(ctx, e.topic(), UserUpdatedType, u); err != nil {
		return fmt.Errorf("publish UserUpdated: %w", err)
	}
	return nil
}

func (e *userEvents) UserDeleted(ctx context.Context, id int64) error {
	payload := struct {
		ID int64 `json:"id"`
	}{ID: id}

	if err := e.bus.Publish(ctx, e.topic(), UserDeletedType, payload); err != nil {
		return fmt.Errorf("publish UserDeleted: %w", err)
	}
	return nil
}
