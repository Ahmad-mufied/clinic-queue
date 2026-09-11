package usecase

import (
	"context"
	"fmt"
	"strings"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

// QueueUseCase implements the inbound.QueueUseCase driving port.
type QueueUseCase struct {
	queueRepo  outbound.QueueRepositoryPort
	doctorRepo outbound.DoctorRepositoryPort
	eventPub   outbound.EventPublisherPort
}

// NewQueueUseCase constructs a new QueueUseCase.
func NewQueueUseCase(
	queueRepo outbound.QueueRepositoryPort,
	doctorRepo outbound.DoctorRepositoryPort,
	eventPub outbound.EventPublisherPort,
) *QueueUseCase {
	return &QueueUseCase{
		queueRepo:  queueRepo,
		doctorRepo: doctorRepo,
		eventPub:   eventPub,
	}
}

var _ inbound.QueueUseCase = (*QueueUseCase)(nil)

const noticeAllDoctorsOffline = "Estimated wait time is currently unavailable because all doctors are offline / on break. Calculation will activate once a doctor starts duty."

// JoinQueue registers a patient in the clinic queue, calculates their wait time, and publishes an update event.
func (u *QueueUseCase) JoinQueue(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
	trimmedName := strings.TrimSpace(patientName)
	if trimmedName == "" {
		return nil, domain.ErrInvalidInput
	}

	// Prevent duplicate active tickets for logged-in user
	if userID != nil && strings.TrimSpace(*userID) != "" {
		activeTicket, err := u.queueRepo.FindActiveTicketByUserID(ctx, *userID)
		if err != nil {
			return nil, fmt.Errorf("check active user ticket: %w", err)
		}
		if activeTicket != nil {
			return nil, domain.ErrActiveTicketExists
		}
	}

	// Prevent duplicate active tickets with identical patient name
	activeByName, err := u.queueRepo.FindActiveTicketByPatientName(ctx, trimmedName)
	if err != nil {
		return nil, fmt.Errorf("check active name ticket: %w", err)
	}
	if activeByName != nil {
		return nil, domain.ErrActiveTicketExists
	}

	// Verify clinic has registered doctors
	allDoctors, err := u.doctorRepo.GetAllDoctorsWithSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all doctors: %w", err)
	}
	if len(allDoctors) == 0 {
		return nil, domain.ErrNoDoctorsAvailable
	}

	// Query currently online doctors
	activeDoctors, err := u.doctorRepo.GetActiveDoctors(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active doctors: %w", err)
	}

	// Get current waiting tickets to determine position
	waitingTickets, err := u.queueRepo.GetWaitingTickets(ctx)
	if err != nil {
		return nil, fmt.Errorf("get waiting tickets: %w", err)
	}

	aheadCount := len(waitingTickets)
	targetPosition := aheadCount + 1

	estimatedWaitTime, notice, err := u.resolveWaitEstimate(activeDoctors, targetPosition)
	if err != nil {
		return nil, err
	}

	queueNumber, err := u.queueRepo.GetNextQueueNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate next queue number: %w", err)
	}

	newTicket := &domain.QueueTicket{
		UserID:                   userID,
		PatientName:              trimmedName,
		QueueNumber:              queueNumber,
		Status:                   domain.TicketStatusWaiting,
		PositionInQueue:          targetPosition,
		AheadCount:               aheadCount,
		EstimatedWaitTimeMinutes: estimatedWaitTime,
		Notice:                   notice,
	}

	createdTicket, err := u.queueRepo.CreateTicket(ctx, newTicket)
	if err != nil {
		return nil, fmt.Errorf("create queue ticket: %w", err)
	}

	createdTicket.PositionInQueue = targetPosition
	createdTicket.AheadCount = aheadCount
	createdTicket.EstimatedWaitTimeMinutes = estimatedWaitTime
	createdTicket.Notice = notice

	// Publish real-time event
	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_JOINED", createdTicket)
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"action":       "QUEUE_JOINED",
			"ticket_id":    createdTicket.ID,
			"queue_number": createdTicket.QueueNumber,
			"patient_name": createdTicket.PatientName,
		})
	}

	return createdTicket, nil
}

// GetMyTicket retrieves the active ticket for a user with updated dynamic wait estimation.
func (u *QueueUseCase) GetMyTicket(ctx context.Context, userID string) (*domain.QueueTicket, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, domain.ErrInvalidInput
	}

	ticket, err := u.queueRepo.FindActiveTicketByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find active ticket: %w", err)
	}
	if ticket == nil {
		return nil, domain.ErrTicketNotFound
	}

	if ticket.Status != domain.TicketStatusWaiting {
		return ticket, nil
	}

	aheadCount, err := u.queueRepo.CountWaitingAhead(ctx, ticket.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("count waiting ahead: %w", err)
	}

	position := aheadCount + 1
	ticket.PositionInQueue = position
	ticket.AheadCount = aheadCount

	activeDoctors, err := u.doctorRepo.GetActiveDoctors(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active doctors: %w", err)
	}

	ticket.EstimatedWaitTimeMinutes, ticket.Notice, _ = u.resolveWaitEstimate(activeDoctors, position)
	return ticket, nil
}

// CancelTicket cancels a waiting ticket for a patient or administrator.
func (u *QueueUseCase) CancelTicket(ctx context.Context, userID *string, userRole string, ticketID string, reason string) (*domain.QueueTicket, error) {
	trimmedTicketID := strings.TrimSpace(ticketID)
	var trimmedUID string
	if userID != nil {
		trimmedUID = strings.TrimSpace(*userID)
	}

	var targetTicket *domain.QueueTicket
	var err error

	// If ticketID is provided, look it up by ID
	if trimmedTicketID != "" {
		targetTicket, err = u.queueRepo.FindByID(ctx, trimmedTicketID)
		if err != nil {
			return nil, fmt.Errorf("find ticket by id %s: %w", trimmedTicketID, err)
		}
		if targetTicket == nil {
			return nil, domain.ErrTicketNotFound
		}
	} else if trimmedUID != "" {
		// If ticketID is empty, resolve active ticket for the authenticated user
		targetTicket, err = u.queueRepo.FindActiveTicketByUserID(ctx, trimmedUID)
		if err != nil {
			return nil, fmt.Errorf("find active ticket for user %s: %w", trimmedUID, err)
		}
		if targetTicket == nil {
			return nil, domain.ErrTicketNotFound
		}
	} else {
		return nil, domain.ErrInvalidInput
	}

	// Validate authorization: only admin or the ticket owner can cancel
	if userRole != string(domain.RoleAdmin) {
		var targetUID string
		if targetTicket.UserID != nil {
			targetUID = strings.TrimSpace(*targetTicket.UserID)
		}
		if trimmedUID != targetUID {
			return nil, domain.ErrUnauthorizedTicketAccess
		}
	}

	// Validate ticket status before calling atomic cancel
	if !targetTicket.CanBeCancelled() {
		return nil, domain.ErrTicketCannotBeCancelled
	}

	// Atomically cancel ticket in repository
	cancelledTicket, err := u.queueRepo.CancelTicketAtomically(ctx, targetTicket.ID)
	if err != nil {
		return nil, fmt.Errorf("cancel ticket atomically: %w", err)
	}

	// Publish real-time events to NATS JetStream
	if u.eventPub != nil {
		var cancelledBy *string
		if trimmedUID != "" {
			cancelledBy = &trimmedUID
		}

		_ = u.eventPub.PublishEvent(ctx, "QUEUE_CANCELLED", map[string]any{
			"ticket_id":     cancelledTicket.ID,
			"queue_number":  cancelledTicket.QueueNumber,
			"patient_name":  cancelledTicket.PatientName,
			"user_id":       cancelledTicket.UserID,
			"cancelled_by":  cancelledBy,
			"role":          userRole,
			"reason":        strings.TrimSpace(reason),
		})

		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"action":       "QUEUE_CANCELLED",
			"ticket_id":    cancelledTicket.ID,
			"queue_number": cancelledTicket.QueueNumber,
			"patient_name": cancelledTicket.PatientName,
		})
	}

	return cancelledTicket, nil
}


// GetQueueStatus retrieves the current state of clinic doctors and the public waiting queue.
func (u *QueueUseCase) GetQueueStatus(ctx context.Context) (*domain.QueueStatus, error) {
	doctorsWithSessions, err := u.doctorRepo.GetAllDoctorsWithSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get doctors with sessions: %w", err)
	}

	activeDoctors, err := u.doctorRepo.GetActiveDoctors(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active doctors: %w", err)
	}

	waitingTickets, err := u.queueRepo.GetWaitingTickets(ctx)
	if err != nil {
		return nil, fmt.Errorf("get waiting tickets: %w", err)
	}

	queueList := make([]domain.QueueTicketSummary, len(waitingTickets))
	for i, t := range waitingTickets {
		waitMin, itemNotice, _ := u.resolveWaitEstimate(activeDoctors, i+1)
		if itemNotice != "" {
			itemNotice = "Awaiting doctor availability"
		}

		queueList[i] = domain.QueueTicketSummary{
			QueueNumber:          t.QueueNumber,
			PatientName:          t.PatientName,
			EstimatedWaitMinutes: waitMin,
			Notice:               itemNotice,
		}
	}

	notice := ""
	if len(activeDoctors) == 0 {
		notice = noticeAllDoctorsOffline
	}

	return &domain.QueueStatus{
		OnlineDoctors: doctorsWithSessions,
		TotalWaiting:  len(waitingTickets),
		QueueList:     queueList,
		Notice:        notice,
	}, nil
}

// resolveWaitEstimate computes the estimated wait time in minutes or returns a descriptive notice when all doctors are offline.
func (u *QueueUseCase) resolveWaitEstimate(activeDoctors []*domain.Doctor, position int) (*int, string, error) {
	if len(activeDoctors) == 0 {
		return nil, noticeAllDoctorsOffline, nil
	}

	waitTime, err := domain.CalculateEstimatedWaitingTime(activeDoctors, position)
	if err != nil {
		return nil, "", fmt.Errorf("calculate estimated waiting time: %w", err)
	}

	return &waitTime, "", nil
}
