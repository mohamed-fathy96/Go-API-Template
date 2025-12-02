package kafka

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/garsue/watermillzap"
	"kabsa/internal/config"
	"kabsa/internal/logging"
	"time"
)

type Router struct {
	router *message.Router
}

func NewRouter(
	ctx context.Context,
	cfg config.KafkaConfig,
	baseLogger logging.Logger,
) (*Router, error) {
	if !cfg.Enabled {
		return &Router{router: nil}, nil
	}

	zapLogger := logging.AsZap(baseLogger)
	wmlogger := watermillzap.NewLogger(zapLogger)

	router, err := message.NewRouter(message.RouterConfig{}, wmlogger)
	if err != nil {
		return nil, fmt.Errorf("create watermill router: %w", err)
	}

	// Subscriber config
	subCfg := kafka.SubscriberConfig{
		Brokers:       cfg.Brokers,
		Unmarshaler:   kafka.DefaultMarshaler{},
		ConsumerGroup: cfg.GroupID,
		InitializeTopicDetails: &sarama.TopicDetail{
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
		NackResendSleep:     5 * time.Second,
		ReconnectRetrySleep: 10 * time.Second,
	}

	subscriber, err := kafka.NewSubscriber(subCfg, wmlogger)
	if err != nil {
		return nil, fmt.Errorf("create kafka subscriber: %w", err)
	}

	// Example: register a handler for users topic (you can refactor into separate file)
	usersTopic := cfg.TopicPrefix + "users"

	router.AddHandler(
		"user-events-handler", // handler name
		usersTopic,            // input topic
		subscriber,
		"",  // no output topic, we're just handling side-effects
		nil, // no publisher (no out topic)
		func(msg *message.Message) ([]*message.Message, error) {
			// TODO: decode Envelope + dispatch to Go service or log
			baseLogger.Info("received message",
				"topic", usersTopic,
				"uuid", msg.UUID,
			)
			return nil, nil
		},
	)

	return &Router{router: router}, nil
}

func (r *Router) Run(ctx context.Context) error {
	if r.router == nil {
		return nil // Kafka disabled
	}
	return r.router.Run(ctx)
}

func (r *Router) Close(ctx context.Context) error {
	if r.router == nil {
		return nil
	}
	_ = r.router.Close()
	return nil
}
