package nats

import (
	"context"
	"math"
	"testing"
)

func TestNATSEventPublisher_PublishEvent_NoConn(t *testing.T) {
	pub := NewNATSEventPublisher(nil, nil)

	// Valid payload with nil connection returns nil (noop)
	err := pub.PublishEvent(context.Background(), "QUEUE_JOINED", map[string]string{"ticket": "A-01"})
	if err != nil {
		t.Errorf("expected nil error on nil conn, got %v", err)
	}

	// Invalid payload (unmarshallable) returns error
	err = pub.PublishEvent(context.Background(), "INVALID", math.NaN())
	if err == nil {
		t.Error("expected error for unmarshallable payload, got nil")
	}

	// Close on nil connection does not error
	if err := pub.Close(); err != nil {
		t.Errorf("expected nil from Close on nil conn, got %v", err)
	}
}
