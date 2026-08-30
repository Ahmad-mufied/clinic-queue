package inbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// JoinQueueRequest represents the HTTP request payload to register for the queue.
type JoinQueueRequest struct {
	PatientName string `json:"patient_name"`
}

// JoinQueueResponse represents the HTTP response payload containing the generated ticket.
type JoinQueueResponse struct {
	Ticket *domain.QueueTicket `json:"ticket"`
}

// QueueUseCase defines the driving/inbound port for patient queue management and wait time estimation.
type QueueUseCase interface {
	// JoinQueue registers a patient into the queue, computes their wait estimation, and publishes an update event.
	JoinQueue(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error)

	// GetMyTicket retrieves the active ticket for a logged-in patient with recalculated wait times.
	GetMyTicket(ctx context.Context, userID string) (*domain.QueueTicket, error)

	// GetQueueStatus retrieves the overall public queue status and active doctor availability.
	GetQueueStatus(ctx context.Context) (*domain.QueueStatus, error)
}
