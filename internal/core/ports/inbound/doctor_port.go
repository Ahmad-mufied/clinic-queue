package inbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// ToggleShiftRequest defines the request body payload to toggle doctor shift status.
type ToggleShiftRequest struct {
	IsOnline bool `json:"is_online"`
}

// DoctorUseCase defines the driving/inbound port for doctor shift management, consultation lifecycle, and workspace monitoring.
type DoctorUseCase interface {
	// ToggleStatus updates the doctor's online availability and broadcasts updated doctor counts.
	ToggleStatus(ctx context.Context, doctorID int, isOnline bool) (*domain.DoctorShiftResponse, error)

	// CallNextPatient atomically pops the next waiting patient from the queue and starts a consultation session.
	CallNextPatient(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)

	// FinishConsultation closes the active consultation session and recalculates public queue wait times.
	FinishConsultation(ctx context.Context, doctorID int) (*domain.ConsultationFinishResponse, error)

	// GetWorkspace retrieves the complete workspace state including room status and active session timer.
	GetWorkspace(ctx context.Context, doctorID int) (*domain.DoctorWorkspace, error)
}
