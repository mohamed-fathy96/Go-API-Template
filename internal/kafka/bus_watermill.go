package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/garsue/watermillzap"
	"github.com/google/uuid"
	"kabsa/internal/config"
	"kabsa/internal/logging"
	"time"
)

type watermillBus struct {
	publisher message.Publisher
	logger    logging.Logger
}

func NewBus(cfg config.KafkaConfig, baseLogger logging.Logger) (Bus, func(ctx context.Context) error, error) {
	if !cfg.Enabled {
		// Return a no-op bus for environments without Kafka
		return &noopBus{}, func(ctx context.Context) error { return nil }, nil
	}

	// Build a Zap logger for Watermill from your logging system
	zapLogger := logging.AsZap(baseLogger) // adapt this helper to your logging package

	wmlogger := watermillzap.NewLogger(zapLogger)

	marshaler := kafka.DefaultMarshaler{}

	// Publisher config
	pubCfg := kafka.PublisherConfig{
		Brokers:   cfg.Brokers,
		Marshaler: marshaler,
		// You can tweak Sarama config here if needed:
		OverwriteSaramaConfig: func() *sarama.Config {
			c := kafka.DefaultSaramaSyncPublisherConfig()
			c.ClientID = cfg.ClientID
			// TODO: add TLS/SASL if needed
			return c
		}(),
	}

	publisher, err := kafka.NewPublisher(pubCfg, wmlogger)
	if err != nil {
		return nil, nil, fmt.Errorf("create kafka publisher: %w", err)
	}

	bus := &watermillBus{
		publisher: publisher,
		logger:    baseLogger.With("component", "kafka_bus"),
	}

	// Close function for graceful shutdown
	closeFn := func(ctx context.Context) error {
		return publisher.Close()
	}

	return bus, closeFn, nil
}

func (b *watermillBus) Publish(ctx context.Context, topic string, msgType string, payload any) error {
	// 1) Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// 2) Wrap into envelope
	env := Envelope{
		MessageID:  uuid.NewString(),
		Type:       msgType,
		OccurredAt: time.Now().UTC(),
		Payload:    payloadBytes,
	}

	// TODO: optionally extract correlation ID from context
	// if cid := correlation.FromContext(ctx); cid != "" {
	//     env.CorrelationID = cid
	// }

	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	msg := message.NewMessage(env.MessageID, body)

	// Optional: set headers (correlation id, trace id, etc.)
	if env.CorrelationID != "" {
		msg.Metadata.Set("correlationId", env.CorrelationID)
	}

	if err := b.publisher.Publish(topic, msg); err != nil {
		b.logger.Error("failed to publish kafka message",
			"topic", topic,
			"type", msgType,
			"error", err,
		)
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// No-op implementation when Kafka is disabled.
type noopBus struct{}

func (*noopBus) Publish(ctx context.Context, topic string, msgType string, payload any) error {
	return nil
}
