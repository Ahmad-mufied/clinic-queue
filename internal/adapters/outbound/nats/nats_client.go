package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamClinicEvents = "CLINIC_EVENTS"
	SubjectClinicWild  = "clinic.>"
)

// NewNATSClient connects to the NATS server and ensures the JetStream stream CLINIC_EVENTS exists.
func NewNATSClient(natsURL string) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(
		natsURL,
		nats.Name("SmartClinicAPI"),
		nats.Timeout(5*time.Second),
		nats.ReconnectWait(1*time.Second),
		nats.MaxReconnects(10),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to nats at %s: %w", natsURL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("initialize jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure CLINIC_EVENTS stream exists
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      StreamClinicEvents,
		Subjects:  []string{SubjectClinicWild},
		Storage:   jetstream.MemoryStorage,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    1 * time.Hour,
	})
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("create or update stream %s: %w", StreamClinicEvents, err)
	}

	return nc, js, nil
}
