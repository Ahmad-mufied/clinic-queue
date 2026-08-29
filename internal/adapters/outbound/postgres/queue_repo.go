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

// QueueRepo implements the outbound.QueueRepositoryPort interface using PostgreSQL 18 via pgx/v5.
type QueueRepo struct {
	pool *pgxpool.Pool
}

// NewQueueRepo constructs a new QueueRepo instance.
func NewQueueRepo(pool *pgxpool.Pool) *QueueRepo {
	return &QueueRepo{pool: pool}
}

var _ outbound.QueueRepositoryPort = (*QueueRepo)(nil)

// CreateTicket inserts a new queue ticket record into PostgreSQL.
func (r *QueueRepo) CreateTicket(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
	query := `
		INSERT INTO queue_tickets (user_id, patient_name, queue_number, status, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		ticket.UserID,
		ticket.PatientName,
		ticket.QueueNumber,
		string(ticket.Status),
	).Scan(&ticket.ID, &ticket.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert queue ticket %s: %w", ticket.QueueNumber, err)
	}

	return ticket, nil
}

// FindActiveTicketByUserID finds an active ticket (WAITING or IN_CONSULTATION) for a specific user ID.
func (r *QueueRepo) FindActiveTicketByUserID(ctx context.Context, userID int) (*domain.QueueTicket, error) {
	query := `
		SELECT id, user_id, patient_name, queue_number, status, created_at, called_at, finished_at
		FROM queue_tickets
		WHERE user_id = $1 AND status IN ('WAITING', 'IN_CONSULTATION')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var ticket domain.QueueTicket
	var statusStr string
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.PatientName,
		&ticket.QueueNumber,
		&statusStr,
		&ticket.CreatedAt,
		&ticket.CalledAt,
		&ticket.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query active ticket for user %d: %w", userID, err)
	}

	ticket.Status = domain.TicketStatus(statusStr)
	return &ticket, nil
}

// FindActiveTicketByPatientName finds an active ticket (WAITING or IN_CONSULTATION) by patient name.
func (r *QueueRepo) FindActiveTicketByPatientName(ctx context.Context, patientName string) (*domain.QueueTicket, error) {
	query := `
		SELECT id, user_id, patient_name, queue_number, status, created_at, called_at, finished_at
		FROM queue_tickets
		WHERE LOWER(patient_name) = LOWER($1) AND status IN ('WAITING', 'IN_CONSULTATION')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var ticket domain.QueueTicket
	var statusStr string
	err := r.pool.QueryRow(ctx, query, patientName).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.PatientName,
		&ticket.QueueNumber,
		&statusStr,
		&ticket.CreatedAt,
		&ticket.CalledAt,
		&ticket.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query active ticket for patient name %s: %w", patientName, err)
	}

	ticket.Status = domain.TicketStatus(statusStr)
	return &ticket, nil
}

// FindByID finds a queue ticket by its primary key ID.
func (r *QueueRepo) FindByID(ctx context.Context, id int) (*domain.QueueTicket, error) {
	query := `
		SELECT id, user_id, patient_name, queue_number, status, created_at, called_at, finished_at
		FROM queue_tickets
		WHERE id = $1
	`

	var ticket domain.QueueTicket
	var statusStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.UserID,
		&ticket.PatientName,
		&ticket.QueueNumber,
		&statusStr,
		&ticket.CreatedAt,
		&ticket.CalledAt,
		&ticket.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query queue ticket by id %d: %w", id, err)
	}

	ticket.Status = domain.TicketStatus(statusStr)
	return &ticket, nil
}

// GetWaitingTickets retrieves all tickets currently in WAITING status ordered chronologically.
func (r *QueueRepo) GetWaitingTickets(ctx context.Context) ([]*domain.QueueTicket, error) {
	query := `
		SELECT id, user_id, patient_name, queue_number, status, created_at, called_at, finished_at
		FROM queue_tickets
		WHERE status = 'WAITING'
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query waiting tickets: %w", err)
	}
	defer rows.Close()

	var tickets []*domain.QueueTicket
	for rows.Next() {
		var ticket domain.QueueTicket
		var statusStr string
		if err := rows.Scan(
			&ticket.ID,
			&ticket.UserID,
			&ticket.PatientName,
			&ticket.QueueNumber,
			&statusStr,
			&ticket.CreatedAt,
			&ticket.CalledAt,
			&ticket.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan waiting ticket row: %w", err)
		}
		ticket.Status = domain.TicketStatus(statusStr)
		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error for waiting tickets: %w", err)
	}

	return tickets, nil
}

// CountWaitingAhead counts how many WAITING tickets were created strictly before the specified timestamp.
func (r *QueueRepo) CountWaitingAhead(ctx context.Context, createdAt time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM queue_tickets
		WHERE status = 'WAITING' AND created_at < $1
	`

	var count int
	err := r.pool.QueryRow(ctx, query, createdAt).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count waiting tickets ahead of %v: %w", createdAt, err)
	}

	return count, nil
}

// GetNextQueueNumber generates the next formatted queue number sequence (e.g. A-01, A-02).
func (r *QueueRepo) GetNextQueueNumber(ctx context.Context) (string, error) {
	query := `
		SELECT COUNT(*)
		FROM queue_tickets
		WHERE created_at >= CURRENT_DATE
	`

	var count int
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("get daily count for queue number: %w", err)
	}

	return fmt.Sprintf("A-%02d", count+1), nil
}
