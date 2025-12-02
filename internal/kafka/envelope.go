package kafka

import (
	"encoding/json"
	"time"
)

type Envelope struct {
	MessageID     string          `json:"messageId"`
	CorrelationID string          `json:"correlationId,omitempty"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Payload       json.RawMessage `json:"payload"`
}
