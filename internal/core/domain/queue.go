package domain

import (
	"time"
)

// TicketStatus represents the lifecycle state of a queue ticket.
type TicketStatus string

const (
	TicketStatusWaiting        TicketStatus = "WAITING"
	TicketStatusInConsultation TicketStatus = "IN_CONSULTATION"
	TicketStatusCompleted      TicketStatus = "COMPLETED"
	TicketStatusCancelled      TicketStatus = "CANCELLED"
)

// IsValid checks whether the ticket status is one of the valid enum values.
func (s TicketStatus) IsValid() bool {
	switch s {
	case TicketStatusWaiting, TicketStatusInConsultation, TicketStatusCompleted, TicketStatusCancelled:
		return true
	default:
		return false
	}
}

// DoctorStatus represents the operational state of a doctor.
type DoctorStatus string

const (
	DoctorStatusAvailable      DoctorStatus = "AVAILABLE"
	DoctorStatusInConsultation DoctorStatus = "IN_CONSULTATION"
	DoctorStatusOffline        DoctorStatus = "OFFLINE"
)

// IsValid checks whether the doctor status is valid.
func (s DoctorStatus) IsValid() bool {
	switch s {
	case DoctorStatusAvailable, DoctorStatusInConsultation, DoctorStatusOffline:
		return true
	default:
		return false
	}
}

// ActiveConsultation represents an ongoing consultation session with a patient.
type ActiveConsultation struct {
	PatientName string `json:"patient_name"`
	ElapsedTime int    `json:"elapsed_time"` // in minutes
}

// RemainingTime calculates the remaining minutes in the consultation session.
// It clamps negative values to 0 if consultation time exceeds the doctor's average.
func (ac *ActiveConsultation) RemainingTime(avgConsultationTime int) int {
	if ac == nil {
		return 0
	}
	return max(0, avgConsultationTime-ac.ElapsedTime)
}

// Doctor represents a medical doctor with consultation parameters and active session.
type Doctor struct {
	ID                  int                 `json:"id"`
	Name                string              `json:"name"`
	AvgConsultationTime int                 `json:"avg_consultation_time"` // in minutes
	IsOnline            bool                `json:"is_online"`
	CurrentSession      *ActiveConsultation `json:"current_session,omitempty"`
}

// NewDoctor creates and validates a new Doctor instance.
func NewDoctor(id int, name string, avgConsultationTime int, isOnline bool, currentSession *ActiveConsultation) (*Doctor, error) {
	if avgConsultationTime <= 0 {
		return nil, ErrInvalidConsultationTime
	}

	return &Doctor{
		ID:                  id,
		Name:                name,
		AvgConsultationTime: avgConsultationTime,
		IsOnline:            isOnline,
		CurrentSession:      currentSession,
	}, nil
}

// DoctorAvailability represents doctor information formatted for public queue status display.
type DoctorAvailability struct {
	ID                         int          `json:"id"`
	Name                       string       `json:"name"`
	AvgConsultationTimeMinutes int          `json:"avg_time"`
	IsOnline                   bool         `json:"is_online"`
	Status                     DoctorStatus `json:"status"`
	CurrentPatientName         string       `json:"current_patient,omitempty"`
	ElapsedMinutes             int          `json:"elapsed_minutes,omitempty"`
}

// QueueTicket represents a walk-in patient's digital queue ticket.
type QueueTicket struct {
	ID                       int          `json:"id"`
	UserID                   *int         `json:"user_id,omitempty"`
	PatientName              string       `json:"patient_name"`
	QueueNumber              string       `json:"queue_number"`
	Status                   TicketStatus `json:"status"`
	PositionInQueue          int          `json:"position_in_queue"`
	AheadCount               int          `json:"ahead_count"`
	EstimatedWaitTimeMinutes *int         `json:"estimated_wait_time_minutes"`
	Notice                   string       `json:"notice,omitempty"`
	CreatedAt                time.Time    `json:"created_at"`
	CalledAt                 *time.Time   `json:"called_at,omitempty"`
	FinishedAt               *time.Time   `json:"finished_at,omitempty"`
}

// QueueTicketSummary represents a summarized queue item for the public queue status list.
type QueueTicketSummary struct {
	QueueNumber          string `json:"queue_number"`
	PatientName          string `json:"patient_name"`
	EstimatedWaitMinutes *int   `json:"estimated_wait_minutes"`
	Notice               string `json:"notice,omitempty"`
}

// QueueStatus represents the overall clinic queue and doctor availability status.
type QueueStatus struct {
	OnlineDoctors []DoctorAvailability `json:"online_doctors"`
	TotalWaiting  int                  `json:"total_waiting"`
	QueueList     []QueueTicketSummary `json:"queue_list"`
	Notice        string               `json:"notice,omitempty"`
}
