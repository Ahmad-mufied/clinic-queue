package outbound

import (
	"context"
	"time"

	"clinic-queue/internal/core/domain"
)

// QueueRepositoryPort defines the driven/outbound SPI interface for queue ticket persistence.
type QueueRepositoryPort interface {
	// CreateTicket inserts a new queue ticket record and returns the persisted entity.
	CreateTicket(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error)

	// FindActiveTicketByUserID finds an active ticket (WAITING or IN_CONSULTATION) for a specific user ID.
	FindActiveTicketByUserID(ctx context.Context, userID int) (*domain.QueueTicket, error)

	// FindActiveTicketByPatientName finds an active ticket (WAITING or IN_CONSULTATION) by patient name.
	FindActiveTicketByPatientName(ctx context.Context, patientName string) (*domain.QueueTicket, error)

	// FindByID finds a queue ticket by its primary key ID.
	FindByID(ctx context.Context, id int) (*domain.QueueTicket, error)

	// GetWaitingTickets retrieves all tickets currently in WAITING status ordered chronologically.
	GetWaitingTickets(ctx context.Context) ([]*domain.QueueTicket, error)

	// CountWaitingAhead counts how many WAITING tickets were created strictly before the specified timestamp.
	CountWaitingAhead(ctx context.Context, createdAt time.Time) (int, error)

	// GetNextQueueNumber generates the next formatted queue number sequence (e.g. A-01, A-02).
	GetNextQueueNumber(ctx context.Context) (string, error)
}
