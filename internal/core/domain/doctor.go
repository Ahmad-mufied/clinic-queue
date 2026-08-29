package domain

import (
	"math"
	"time"
)

// ConsultationTicket represents the ticket information embedded in a consultation session.
type ConsultationTicket struct {
	ID          int          `json:"id"`
	QueueNumber string       `json:"queue_number"`
	PatientName string       `json:"patient_name"`
	Status      TicketStatus `json:"status"`
}

// ConsultationSession represents an examination session conducted by a doctor for a ticket.
type ConsultationSession struct {
	ID          int                 `json:"session_id,omitempty"`
	DoctorID    int                 `json:"doctor_id"`
	TicketID    int                 `json:"ticket_id,omitempty"`
	PatientName string              `json:"patient_name,omitempty"`
	Ticket      *ConsultationTicket `json:"ticket,omitempty"`
	StartedAt   time.Time           `json:"started_at"`
	FinishedAt  *time.Time          `json:"finished_at,omitempty"`
	IsActive    bool                `json:"is_active"`
}

// DurationMinutes returns the elapsed consultation duration in minutes rounded to 1 decimal place.
func (cs *ConsultationSession) DurationMinutes() float64 {
	if cs == nil {
		return 0
	}
	if cs.FinishedAt == nil {
		return CalculateDurationMinutes(cs.StartedAt, time.Now())
	}
	return CalculateDurationMinutes(cs.StartedAt, *cs.FinishedAt)
}

// CalculateDurationMinutes calculates the difference in minutes between two timestamps rounded to 1 decimal place.
func CalculateDurationMinutes(start, finish time.Time) float64 {
	diff := finish.Sub(start).Seconds()
	if diff < 0 {
		return 0
	}
	minutes := diff / 60.0
	return math.Round(minutes*10) / 10
}

// DetermineDoctorStatus calculates the operational status of a doctor based on shift and session state.
func DetermineDoctorStatus(isOnline bool, hasActiveSession bool) DoctorStatus {
	if !isOnline {
		return DoctorStatusOffline
	}
	if hasActiveSession {
		return DoctorStatusInConsultation
	}
	return DoctorStatusAvailable
}

// DoctorShiftResponse defines the response payload for shift status updates.
type DoctorShiftResponse struct {
	DoctorID int          `json:"doctor_id"`
	Name     string       `json:"name"`
	IsOnline bool         `json:"is_online"`
	Status   DoctorStatus `json:"status"`
}

// ConsultationFinishResponse defines the response payload returned upon completing a consultation.
type ConsultationFinishResponse struct {
	SessionID             int          `json:"session_id"`
	PatientName           string       `json:"patient_name"`
	ActualDurationMinutes float64      `json:"actual_duration_minutes"`
	FinishedAt            time.Time    `json:"finished_at"`
	DoctorStatus          DoctorStatus `json:"doctor_status"`
}

// DoctorWorkspace defines the complete workspace snapshot for an authenticated doctor.
type DoctorWorkspace struct {
	DoctorID            int                  `json:"doctor_id"`
	DoctorName          string               `json:"doctor_name"`
	AvgConsultationTime int                  `json:"avg_consultation_time"`
	IsOnline            bool                 `json:"is_online"`
	Status              DoctorStatus         `json:"status"`
	ActiveSession       *ConsultationSession `json:"active_session,omitempty"`
}
