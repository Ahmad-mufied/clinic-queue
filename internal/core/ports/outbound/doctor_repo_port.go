package outbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// DoctorRepositoryPort defines the driven/outbound SPI interface for doctor master data and session querying.
type DoctorRepositoryPort interface {
	// GetActiveDoctors retrieves all online doctors along with their active consultation session state.
	GetActiveDoctors(ctx context.Context) ([]*domain.Doctor, error)

	// GetAllDoctorsWithSessions retrieves all doctors (both online and offline) with formatted availability status.
	GetAllDoctorsWithSessions(ctx context.Context) ([]domain.DoctorAvailability, error)

	// GetDoctorByID retrieves a doctor by their primary key ID.
	GetDoctorByID(ctx context.Context, id int) (*domain.Doctor, error)

	// UpdateOnlineStatus updates the is_online status of a doctor.
	UpdateOnlineStatus(ctx context.Context, doctorID int, isOnline bool) error

	// GetActiveSessionByDoctorID retrieves the active consultation session for a given doctor.
	GetActiveSessionByDoctorID(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)

	// UpdateDoctorAvgTime updates the configured average consultation duration for a doctor.
	UpdateDoctorAvgTime(ctx context.Context, doctorID int, avgTime int) error
}

