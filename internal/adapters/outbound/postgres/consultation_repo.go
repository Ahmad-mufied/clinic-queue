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

// ConsultationRepo implements outbound.ConsultationRepositoryPort using PostgreSQL 18.
type ConsultationRepo struct {
	pool *pgxpool.Pool
}

// NewConsultationRepo constructs a new ConsultationRepo instance.
func NewConsultationRepo(pool *pgxpool.Pool) *ConsultationRepo {
	return &ConsultationRepo{pool: pool}
}

var _ outbound.ConsultationRepositoryPort = (*ConsultationRepo)(nil)

// CallNextTicketAtomically acquires the next waiting patient ticket via row-level lock (FOR UPDATE SKIP LOCKED),
// updates the ticket status to IN_CONSULTATION, and creates a consultation session in a single transaction.
func (r *ConsultationRepo) CallNextTicketAtomically(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock and fetch the earliest waiting ticket
	selectQuery := `
		SELECT id, queue_number, patient_name, status, created_at
		FROM queue_tickets
		WHERE status = 'WAITING'
		ORDER BY id ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var (
		ticketID    string
		queueNumber string
		patientName string
		statusStr   string
		createdAt   time.Time
	)

	err = tx.QueryRow(ctx, selectQuery).Scan(&ticketID, &queueNumber, &patientName, &statusStr, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Queue is empty
	}
	if err != nil {
		return nil, fmt.Errorf("lock earliest waiting ticket: %w", err)
	}

	// 2. Mark ticket as IN_CONSULTATION
	updateTicketQuery := `
		UPDATE queue_tickets
		SET status = 'IN_CONSULTATION', called_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, updateTicketQuery, ticketID); err != nil {
		return nil, fmt.Errorf("update ticket status to in_consultation: %w", err)
	}

	// 3. Insert active consultation session record
	insertSessionQuery := `
		INSERT INTO consultation_sessions (doctor_id, ticket_id, patient_name, started_at, is_active)
		VALUES ($1, $2, $3, NOW(), TRUE)
		RETURNING id, started_at
	`
	var (
		sessionID string
		startedAt time.Time
	)

	if err := tx.QueryRow(ctx, insertSessionQuery, doctorID, ticketID, patientName).Scan(&sessionID, &startedAt); err != nil {
		return nil, fmt.Errorf("insert consultation session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit call-next transaction: %w", err)
	}

	return &domain.ConsultationSession{
		ID:          sessionID,
		DoctorID:    doctorID,
		TicketID:    ticketID,
		PatientName: patientName,
		Ticket: &domain.ConsultationTicket{
			ID:          ticketID,
			QueueNumber: queueNumber,
			PatientName: patientName,
			Status:      domain.TicketStatusInConsultation,
		},
		StartedAt: startedAt,
		IsActive:  true,
	}, nil
}

// FinishActiveSession closes the doctor's active session and marks the corresponding ticket COMPLETED in a single transaction.
func (r *ConsultationRepo) FinishActiveSession(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Close active consultation session
	updateSessionQuery := `
		UPDATE consultation_sessions
		SET is_active = FALSE, finished_at = NOW()
		WHERE doctor_id = $1 AND is_active = TRUE
		RETURNING id, ticket_id, patient_name, started_at, finished_at
	`

	var (
		sessionID   string
		ticketID    string
		patientName string
		startedAt   time.Time
		finishedAt  time.Time
	)

	err = tx.QueryRow(ctx, updateSessionQuery, doctorID).Scan(&sessionID, &ticketID, &patientName, &startedAt, &finishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No active session to finish
	}
	if err != nil {
		return nil, fmt.Errorf("update consultation session to finished: %w", err)
	}

	// 2. Mark ticket as COMPLETED
	updateTicketQuery := `
		UPDATE queue_tickets
		SET status = 'COMPLETED', finished_at = $1
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateTicketQuery, finishedAt, ticketID); err != nil {
		return nil, fmt.Errorf("update ticket status to completed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit finish transaction: %w", err)
	}

	return &domain.ConsultationSession{
		ID:          sessionID,
		DoctorID:    doctorID,
		TicketID:    ticketID,
		PatientName: patientName,
		StartedAt:   startedAt,
		FinishedAt:  &finishedAt,
		IsActive:    false,
	}, nil
}

// GetActiveSessionByDoctorID retrieves the active consultation session for a given doctor.
func (r *ConsultationRepo) GetActiveSessionByDoctorID(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	query := `
		SELECT cs.id, cs.doctor_id, cs.ticket_id, cs.patient_name, cs.started_at, cs.finished_at, cs.is_active,
		       qt.queue_number, qt.status
		FROM consultation_sessions cs
		LEFT JOIN queue_tickets qt ON cs.ticket_id = qt.id
		WHERE cs.doctor_id = $1 AND cs.is_active = true
		LIMIT 1
	`
	var (
		sessionID    string
		docID        string
		ticketID     string
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
		return nil, fmt.Errorf("query active session by doctor id %s: %w", doctorID, err)
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
