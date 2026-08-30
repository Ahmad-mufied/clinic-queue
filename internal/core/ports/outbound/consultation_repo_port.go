package outbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// ConsultationRepositoryPort defines the driven/outbound SPI interface for consultation session management and atomic patient admissions.
type ConsultationRepositoryPort interface {
	// CallNextTicketAtomically acquires the earliest waiting ticket using SELECT FOR UPDATE SKIP LOCKED,
	// marks it IN_CONSULTATION, and creates an active consultation session record within a single atomic transaction.
	CallNextTicketAtomically(ctx context.Context, doctorID string) (*domain.ConsultationSession, error)

	// FinishActiveSession completes the active consultation session and marks the corresponding ticket as COMPLETED.
	FinishActiveSession(ctx context.Context, doctorID string) (*domain.ConsultationSession, error)

	// GetActiveSessionByDoctorID retrieves the current active consultation session for a given doctor.
	GetActiveSessionByDoctorID(ctx context.Context, doctorID string) (*domain.ConsultationSession, error)
}
