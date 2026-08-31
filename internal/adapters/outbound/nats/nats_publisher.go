package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATSEventPublisher implements outbound.EventPublisherPort using NATS JetStream.
type NATSEventPublisher struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewNATSEventPublisher constructs a new NATSEventPublisher.
func NewNATSEventPublisher(nc *nats.Conn, js jetstream.JetStream) *NATSEventPublisher {
	return &NATSEventPublisher{
		nc: nc,
		js: js,
	}
}

var _ outbound.EventPublisherPort = (*NATSEventPublisher)(nil)

// EventMessage represents the standard envelope for published clinic events.
type EventMessage struct {
	Type      string                `json:"type"`
	Data      any                   `json:"data"`
	Timestamp time.Time             `json:"timestamp"`
	Metadata  domain.ClientMetadata `json:"metadata,omitempty"`
}

// PublishEvent serializes the event payload to JSON and publishes it to NATS JetStream.
func (p *NATSEventPublisher) PublishEvent(ctx context.Context, eventType string, payload any) error {
	subject := fmt.Sprintf("clinic.events.%s", strings.ToLower(eventType))

	msg := EventMessage{
		Type:      eventType,
		Data:      payload,
		Timestamp: time.Now().UTC(),
	}

	if ctx != nil {
		if meta, ok := domain.GetClientMetadata(ctx); ok {
			msg.Metadata = meta
		}
	}


	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal event %s payload: %w", eventType, err)
	}

	switch {
	case p.js != nil:
		if _, err := p.js.Publish(ctx, subject, data); err != nil {
			return fmt.Errorf("publish event to jetstream subject %s: %w", subject, err)
		}
	case p.nc != nil:
		if err := p.nc.Publish(subject, data); err != nil {
			return fmt.Errorf("publish event to nats subject %s: %w", subject, err)
		}
	}

	return nil
}

// Close drains and closes the NATS client connection.
func (p *NATSEventPublisher) Close() error {
	if p.nc != nil {
		_ = p.nc.Drain()
		p.nc.Close()
	}
	return nil
}
