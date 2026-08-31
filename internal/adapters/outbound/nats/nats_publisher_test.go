package nats

import (
	"context"
	"math"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestNATSEventPublisher_PublishEvent_NoConn(t *testing.T) {
	pub := NewNATSEventPublisher(nil, nil)

	// Valid payload with nil connection returns nil (noop)
	err := pub.PublishEvent(context.Background(), "QUEUE_JOINED", map[string]string{"ticket": "A-01"})
	if err != nil {
		t.Errorf("expected nil error on nil conn, got %v", err)
	}

	// Valid payload with metadata in context
	ctxWithMeta := domain.WithClientMetadata(context.Background(), domain.ClientMetadata{
		ClientIP:  "203.0.113.1",
		UserAgent: "TestAgent/1.0",
		RequestID: "req-xyz-789",
	})
	err = pub.PublishEvent(ctxWithMeta, "AUTH_LOGIN", map[string]string{"user_id": "123"})
	if err != nil {
		t.Errorf("expected nil error on publishing with metadata, got %v", err)
	}

	// Nil context handling
	err = pub.PublishEvent(nil, "QUEUE_JOINED", map[string]string{"ticket": "A-02"})
	if err != nil {
		t.Errorf("expected nil error on nil context, got %v", err)
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

func TestNATSEventPublisher_WithConnection(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("skipping NATS live connection test: nats server not reachable")
		return
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream.New error = %v", err)
	}

	// Ensure test stream exists
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "CLINIC_TEST_PUB_STREAM",
		Subjects: []string{"clinic.events.>"},
	})

	// Test publishing with JetStream
	pub := NewNATSEventPublisher(nc, js)
	err = pub.PublishEvent(ctx, "QUEUE_JOINED", map[string]string{"patient": "Alice"})
	if err != nil {
		t.Errorf("expected nil error on JS publish, got %v", err)
	}

	// Test publishing with standard NATS connection fallback (js = nil)
	pubNC := NewNATSEventPublisher(nc, nil)
	err = pubNC.PublishEvent(ctx, "QUEUE_JOINED", map[string]string{"patient": "Bob"})
	if err != nil {
		t.Errorf("expected nil error on NC publish, got %v", err)
	}

	// Test publishing error on closed connection
	closedNC, err := nats.Connect(nats.DefaultURL)
	if err == nil {
		pubClosedNC := NewNATSEventPublisher(closedNC, nil)
		closedNC.Close()
		err = pubClosedNC.PublishEvent(ctx, "QUEUE_JOINED", map[string]string{"patient": "Error"})
		if err == nil {
			t.Error("expected error on closed NC publish, got nil")
		}
	}

	// Test NewNATSClient
	clientNC, clientJS, err := NewNATSClient(nats.DefaultURL)
	if err != nil {
		t.Errorf("NewNATSClient error = %v", err)
	}
	if clientNC != nil {
		clientNC.Close()
	}
	_ = clientJS

	// Test NewNATSClient with invalid URL using unreachable localhost port (instant connection refused)
	_, _, err = NewNATSClient("nats://127.0.0.1:59999")
	if err == nil {
		t.Error("expected error for invalid NATS URL, got nil")
	}
}

