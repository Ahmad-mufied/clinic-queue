package domain

import (
	"errors"
	"testing"
)

func TestTicketStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status TicketStatus
		want   bool
	}{
		{"Waiting status", TicketStatusWaiting, true},
		{"In consultation status", TicketStatusInConsultation, true},
		{"Completed status", TicketStatusCompleted, true},
		{"Cancelled status", TicketStatusCancelled, true},
		{"Invalid status", TicketStatus("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("TicketStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoctorStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status DoctorStatus
		want   bool
	}{
		{"Available status", DoctorStatusAvailable, true},
		{"In consultation status", DoctorStatusInConsultation, true},
		{"Offline status", DoctorStatusOffline, true},
		{"Invalid status", DoctorStatus("ON_VACATION"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("DoctorStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveConsultation_RemainingTime(t *testing.T) {
	tests := []struct {
		name     string
		session  *ActiveConsultation
		avgTime  int
		wantTime int
	}{
		{
			name:     "Nil session returns 0",
			session:  nil,
			avgTime:  5,
			wantTime: 0,
		},
		{
			name:     "Remaining time when elapsed < avg",
			session:  &ActiveConsultation{PatientName: "Alice", ElapsedTime: 2},
			avgTime:  5,
			wantTime: 3,
		},
		{
			name:     "Remaining time clamped to 0 when elapsed >= avg",
			session:  &ActiveConsultation{PatientName: "Bob", ElapsedTime: 7},
			avgTime:  5,
			wantTime: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.RemainingTime(tt.avgTime); got != tt.wantTime {
				t.Errorf("ActiveConsultation.RemainingTime() = %d, want %d", got, tt.wantTime)
			}
		})
	}
}

func TestNewDoctor(t *testing.T) {
	tests := []struct {
		name        string
		id          int
		docName     string
		avgTime     int
		isOnline    bool
		session     *ActiveConsultation
		wantErr     error
		wantNil     bool
	}{
		{
			name:        "Invalid avg consultation time <= 0",
			id:          1,
			docName:     "Dr. Error",
			avgTime:     0,
			isOnline:    true,
			session:     nil,
			wantErr:     ErrInvalidConsultationTime,
			wantNil:     true,
		},
		{
			name:        "Negative avg consultation time",
			id:          1,
			docName:     "Dr. Error",
			avgTime:     -3,
			isOnline:    true,
			session:     nil,
			wantErr:     ErrInvalidConsultationTime,
			wantNil:     true,
		},
		{
			name:        "Valid doctor creation",
			id:          1,
			docName:     "Dr. Sarah Adams",
			avgTime:     3,
			isOnline:    true,
			session:     &ActiveConsultation{PatientName: "Lucas", ElapsedTime: 1},
			wantErr:     nil,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewDoctor(tt.id, tt.docName, tt.avgTime, tt.isOnline, tt.session)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewDoctor() error = %v, wantErr %v", err, tt.wantErr)
				}
				if (doc == nil) != tt.wantNil {
					t.Errorf("NewDoctor() doc = %v, wantNil %v", doc, tt.wantNil)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if doc == nil || doc.Name != tt.docName || doc.AvgConsultationTime != tt.avgTime || doc.IsOnline != tt.isOnline {
				t.Errorf("NewDoctor() unexpected doctor entity: %+v", doc)
			}
		})
	}
}
