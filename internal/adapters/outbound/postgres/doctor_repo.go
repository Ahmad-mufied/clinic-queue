package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DoctorRepo implements the outbound.DoctorRepositoryPort interface using PostgreSQL 18.
type DoctorRepo struct {
	pool *pgxpool.Pool
}

// NewDoctorRepo constructs a new DoctorRepo instance.
func NewDoctorRepo(pool *pgxpool.Pool) *DoctorRepo {
	return &DoctorRepo{pool: pool}
}

var _ outbound.DoctorRepositoryPort = (*DoctorRepo)(nil)

// GetActiveDoctors retrieves all online doctors with their current consultation session.
func (r *DoctorRepo) GetActiveDoctors(ctx context.Context) ([]*domain.Doctor, error) {
	query := `
		SELECT 
			d.id, 
			d.name, 
			d.avg_consultation_time_min, 
			d.is_online,
			cs.patient_name,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - cs.started_at)) / 60, 0)::INT AS elapsed_minutes
		FROM doctors d
		LEFT JOIN consultation_sessions cs 
			ON d.id = cs.doctor_id AND cs.is_active = true
		WHERE d.is_online = true
		ORDER BY d.id ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active doctors: %w", err)
	}
	defer rows.Close()

	var doctors []*domain.Doctor
	for rows.Next() {
		var (
			id             int
			name           string
			avgTime        int
			isOnline       bool
			patientName    *string
			elapsedMinutes int
		)

		if err := rows.Scan(&id, &name, &avgTime, &isOnline, &patientName, &elapsedMinutes); err != nil {
			return nil, fmt.Errorf("scan active doctor row: %w", err)
		}

		var currentSession *domain.ActiveConsultation
		if patientName != nil {
			currentSession = &domain.ActiveConsultation{
				PatientName: *patientName,
				ElapsedTime: elapsedMinutes,
			}
		}

		doc := &domain.Doctor{
			ID:                  id,
			Name:                name,
			AvgConsultationTime: avgTime,
			IsOnline:            isOnline,
			CurrentSession:      currentSession,
		}
		doctors = append(doctors, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error for active doctors: %w", err)
	}

	return doctors, nil
}

// GetAllDoctorsWithSessions retrieves all doctors with formatted availability status for public queue status display.
func (r *DoctorRepo) GetAllDoctorsWithSessions(ctx context.Context) ([]domain.DoctorAvailability, error) {
	query := `
		SELECT 
			d.id, 
			d.name, 
			d.avg_consultation_time_min, 
			d.is_online,
			cs.patient_name,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - cs.started_at)) / 60, 0)::INT AS elapsed_minutes
		FROM doctors d
		LEFT JOIN consultation_sessions cs 
			ON d.id = cs.doctor_id AND cs.is_active = true
		ORDER BY d.id ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all doctors with sessions: %w", err)
	}
	defer rows.Close()

	var availabilities []domain.DoctorAvailability
	for rows.Next() {
		var (
			id             int
			name           string
			avgTime        int
			isOnline       bool
			patientName    *string
			elapsedMinutes int
		)

		if err := rows.Scan(&id, &name, &avgTime, &isOnline, &patientName, &elapsedMinutes); err != nil {
			return nil, fmt.Errorf("scan doctor availability row: %w", err)
		}

		var (
			status         domain.DoctorStatus
			currentPatient string
		)
		switch {
		case !isOnline:
			status = domain.DoctorStatusOffline
		case patientName != nil:
			status = domain.DoctorStatusInConsultation
			currentPatient = *patientName
		default:
			status = domain.DoctorStatusAvailable
		}

		availabilities = append(availabilities, domain.DoctorAvailability{
			ID:                         id,
			Name:                       name,
			AvgConsultationTimeMinutes: avgTime,
			IsOnline:                   isOnline,
			Status:                     status,
			CurrentPatientName:         currentPatient,
			ElapsedMinutes:             elapsedMinutes,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error for all doctors with sessions: %w", err)
	}

	return availabilities, nil
}

// GetDoctorByID retrieves a single doctor by primary key ID.
func (r *DoctorRepo) GetDoctorByID(ctx context.Context, id int) (*domain.Doctor, error) {
	query := `
		SELECT 
			d.id, 
			d.name, 
			d.avg_consultation_time_min, 
			d.is_online,
			cs.patient_name,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - cs.started_at)) / 60, 0)::INT AS elapsed_minutes
		FROM doctors d
		LEFT JOIN consultation_sessions cs 
			ON d.id = cs.doctor_id AND cs.is_active = true
		WHERE d.id = $1
	`

	var (
		docID          int
		name           string
		avgTime        int
		isOnline       bool
		patientName    *string
		elapsedMinutes int
	)

	err := r.pool.QueryRow(ctx, query, id).Scan(&docID, &name, &avgTime, &isOnline, &patientName, &elapsedMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query doctor by id %d: %w", id, err)
	}

	var currentSession *domain.ActiveConsultation
	if patientName != nil {
		currentSession = &domain.ActiveConsultation{
			PatientName: *patientName,
			ElapsedTime: elapsedMinutes,
		}
	}

	return &domain.Doctor{
		ID:                  docID,
		Name:                name,
		AvgConsultationTime: avgTime,
		IsOnline:            isOnline,
		CurrentSession:      currentSession,
	}, nil
}

// UpdateOnlineStatus updates the is_online status of a doctor.
func (r *DoctorRepo) UpdateOnlineStatus(ctx context.Context, doctorID int, isOnline bool) error {
	query := `UPDATE doctors SET is_online = $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.pool.Exec(ctx, query, isOnline, doctorID)
	if err != nil {
		return fmt.Errorf("update doctor online status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDoctorNotFound
	}
	return nil
}

// GetActiveSessionByDoctorID retrieves the active consultation session for a given doctor.
func (r *DoctorRepo) GetActiveSessionByDoctorID(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
	query := `
		SELECT cs.id, cs.doctor_id, cs.ticket_id, cs.patient_name, cs.started_at, cs.finished_at, cs.is_active,
		       qt.queue_number, qt.status
		FROM consultation_sessions cs
		LEFT JOIN queue_tickets qt ON cs.ticket_id = qt.id
		WHERE cs.doctor_id = $1 AND cs.is_active = true
		LIMIT 1
	`
	var (
		sessionID    int
		docID        int
		ticketID     int
		patientName  string
		startedAt    time.Time
		finishedAt   *time.Time
		isActive     bool
		queueNumber  *string
		ticketStatus *string
	)

	err := r.pool.QueryRow(ctx, query, doctorID).Scan(
		&sessionID, &docID, &ticketID, &patientName, &startedAt, &finishedAt, &isActive,
		&queueNumber, &ticketStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query active session by doctor id %d: %w", doctorID, err)
	}

	var ticket *domain.ConsultationTicket
	if queueNumber != nil && ticketStatus != nil {
		ticket = &domain.ConsultationTicket{
			ID:          ticketID,
			QueueNumber: *queueNumber,
			PatientName: patientName,
			Status:      domain.TicketStatus(*ticketStatus),
		}
	}

	return &domain.ConsultationSession{
		ID:          sessionID,
		DoctorID:    docID,
		TicketID:    ticketID,
		PatientName: patientName,
		Ticket:      ticket,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		IsActive:    isActive,
	}, nil
}
