package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

// DoctorUseCase implements inbound.DoctorUseCase for doctor workspace and shift management.
type DoctorUseCase struct {
	doctorRepo       outbound.DoctorRepositoryPort
	consultationRepo outbound.ConsultationRepositoryPort
	eventPub         outbound.EventPublisherPort
}

// NewDoctorUseCase constructs a new DoctorUseCase instance.
func NewDoctorUseCase(
	doctorRepo outbound.DoctorRepositoryPort,
	consultationRepo outbound.ConsultationRepositoryPort,
	eventPub outbound.EventPublisherPort,
) *DoctorUseCase {
	return &DoctorUseCase{
		doctorRepo:       doctorRepo,
		consultationRepo: consultationRepo,
		eventPub:         eventPub,
	}
}

var _ inbound.DoctorUseCase = (*DoctorUseCase)(nil)

// ToggleStatus updates the doctor's online availability and broadcasts status change events.
func (u *DoctorUseCase) ToggleStatus(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
	if strings.TrimSpace(doctorID) == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := u.doctorRepo.GetDoctorByID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor by id: %w", err)
	}
	if doc == nil {
		return nil, domain.ErrDoctorNotFound
	}

	activeSession, err := u.consultationRepo.GetActiveSessionByDoctorID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("check active session: %w", err)
	}

	if err := u.doctorRepo.UpdateOnlineStatus(ctx, doctorID, isOnline); err != nil {
		return nil, fmt.Errorf("update online status: %w", err)
	}

	status := domain.DetermineDoctorStatus(isOnline, activeSession != nil)

	resp := &domain.DoctorShiftResponse{
		DoctorID: doctorID,
		Name:     doc.Name,
		IsOnline: isOnline,
		Status:   status,
	}

	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "DOCTOR_STATUS_CHANGED", resp)
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"doctor_id":   doctorID,
			"doctor_name": doc.Name,
			"is_online":   isOnline,
			"status":      status,
		})
	}

	return resp, nil
}

// CallNextPatient atomically pops the next waiting patient from the queue and creates a consultation session.
func (u *DoctorUseCase) CallNextPatient(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	if strings.TrimSpace(doctorID) == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := u.doctorRepo.GetDoctorByID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor by id: %w", err)
	}
	if doc == nil {
		return nil, domain.ErrDoctorNotFound
	}

	if !doc.IsOnline {
		return nil, domain.ErrDoctorOffline
	}

	activeSession, err := u.consultationRepo.GetActiveSessionByDoctorID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("check active session: %w", err)
	}
	if activeSession != nil {
		return nil, domain.ErrActiveConsultationExists
	}

	session, err := u.consultationRepo.CallNextTicketAtomically(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("call next ticket atomically: %w", err)
	}
	if session == nil {
		return nil, domain.ErrQueueEmpty
	}

	session.DoctorName = doc.Name

	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "TICKET_CALLED", session)
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"action":      "TICKET_CALLED",
			"session_id":  session.ID,
			"doctor_id":   doctorID,
			"doctor_name": doc.Name,
		})
	}

	return session, nil
}

// FinishConsultation marks the active session completed, computes actual duration, and resets doctor availability.
func (u *DoctorUseCase) FinishConsultation(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
	if strings.TrimSpace(doctorID) == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := u.doctorRepo.GetDoctorByID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor by id: %w", err)
	}
	if doc == nil {
		return nil, domain.ErrDoctorNotFound
	}

	activeSession, err := u.consultationRepo.GetActiveSessionByDoctorID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("check active session: %w", err)
	}
	if activeSession == nil {
		return nil, domain.ErrNoActiveConsultation
	}

	finishedSession, err := u.consultationRepo.FinishActiveSession(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("finish active session: %w", err)
	}
	if finishedSession == nil {
		return nil, domain.ErrNoActiveConsultation
	}

	actualDuration := finishedSession.DurationMinutes()

	docStatus := domain.DoctorStatusAvailable
	if !doc.IsOnline {
		docStatus = domain.DoctorStatusOffline
	}

	finishedAt := time.Now().UTC()
	if finishedSession.FinishedAt != nil {
		finishedAt = *finishedSession.FinishedAt
	}

	resp := &domain.ConsultationFinishResponse{
		SessionID:             finishedSession.ID,
		DoctorID:              doc.ID,
		DoctorName:            doc.Name,
		PatientName:           finishedSession.PatientName,
		ActualDurationMinutes: actualDuration,
		FinishedAt:            finishedAt,
		DoctorStatus:          docStatus,
	}

	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "TICKET_FINISHED", resp)
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"action":      "TICKET_FINISHED",
			"session_id":  resp.SessionID,
			"doctor_id":   doctorID,
			"doctor_name": doc.Name,
		})
	}

	return resp, nil
}

// GetWorkspace retrieves doctor workspace data, including active session and real-time status.
func (u *DoctorUseCase) GetWorkspace(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
	if strings.TrimSpace(doctorID) == "" {
		return nil, domain.ErrInvalidInput
	}

	doc, err := u.doctorRepo.GetDoctorByID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor by id: %w", err)
	}
	if doc == nil {
		return nil, domain.ErrDoctorNotFound
	}

	activeSession, err := u.consultationRepo.GetActiveSessionByDoctorID(ctx, doctorID)
	if err != nil {
		return nil, fmt.Errorf("get active session: %w", err)
	}

	status := domain.DetermineDoctorStatus(doc.IsOnline, activeSession != nil)

	return &domain.DoctorWorkspace{
		DoctorID:            doc.ID,
		DoctorName:          doc.Name,
		AvgConsultationTime: doc.AvgConsultationTime,
		IsOnline:            doc.IsOnline,
		Status:              status,
		ActiveSession:       activeSession,
	}, nil
}
