package outbound

import (
	"context"
)

// EventPublisherPort defines the driven/outbound SPI interface for broadcasting domain events.
type EventPublisherPort interface {
	// PublishEvent serializes and publishes an event to the real-time event broker.
	PublishEvent(ctx context.Context, eventType string, payload any) error

	// Close shuts down the publisher connection gracefully.
	Close() error
}
