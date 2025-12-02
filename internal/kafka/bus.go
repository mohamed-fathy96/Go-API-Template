package kafka

import "context"

type Bus interface {
	Publish(ctx context.Context, topic string, msgType string, payload any) error
}
