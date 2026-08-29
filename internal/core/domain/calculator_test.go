package domain

import (
	"errors"
	"testing"
)

func TestCalculateEstimatedWaitingTime(t *testing.T) {
	tests := []struct {
		name        string
		doctors     []*Doctor
		position    int
		wantMinutes int
		wantErr     error
	}{
		{
			name: "Case Study 1 Q1: 1 Doc (3m), idle, John is 6th",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
			},
			position:    6,
			wantMinutes: 15,
			wantErr:     nil,
		},
		{
			name: "Case Study 1 Q2(a): 1 Doc (3m), Peter in 2m (1m left), 4 waiting ahead (John is 5th)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true, CurrentSession: &ActiveConsultation{PatientName: "Peter", ElapsedTime: 2}},
			},
			position:    5,
			wantMinutes: 13,
			wantErr:     nil,
		},
		{
			name: "Case Study 1 Q2(b): 1 Doc (3m), Peter in 5m (0m left / exceeded), 4 waiting ahead (John is 5th)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true, CurrentSession: &ActiveConsultation{PatientName: "Peter", ElapsedTime: 5}},
			},
			position:    5,
			wantMinutes: 12,
			wantErr:     nil,
		},
		{
			name: "Case Study 2 Q1: Doc A (3m), Doc B (4m), both idle, John is 11th",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
				{ID: 2, Name: "Doctor B", AvgConsultationTime: 4, IsOnline: true, CurrentSession: nil},
			},
			position:    11,
			wantMinutes: 16,
			wantErr:     nil,
		},
		{
			name: "Case Study 2 Q2: Doc A idle, Doc B with Lucas (2m elapsed of 4m), 9 waiting ahead (John is 10th)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
				{ID: 2, Name: "Doctor B", AvgConsultationTime: 4, IsOnline: true, CurrentSession: &ActiveConsultation{PatientName: "Lucas", ElapsedTime: 2}},
			},
			position:    10,
			wantMinutes: 15,
			wantErr:     nil,
		},
		{
			name: "Edge Case: Immediate entry (Position 1)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
			},
			position:    1,
			wantMinutes: 0,
			wantErr:     nil,
		},
		{
			name:        "Error: Empty doctors list",
			doctors:     []*Doctor{},
			position:    5,
			wantMinutes: 0,
			wantErr:     ErrEmptyDoctors,
		},
		{
			name: "Error: Invalid position (0)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
			},
			position:    0,
			wantMinutes: 0,
			wantErr:     ErrInvalidPosition,
		},
		{
			name: "Error: Invalid position (-1)",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
			},
			position:    -1,
			wantMinutes: 0,
			wantErr:     ErrInvalidPosition,
		},
		{
			name: "Error: Nil doctor in slice",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true, CurrentSession: nil},
				nil,
			},
			position:    3,
			wantMinutes: 0,
			wantErr:     ErrNilDoctor,
		},
		{
			name: "Error: Doctor with non-positive AvgConsultationTime",
			doctors: []*Doctor{
				{ID: 1, Name: "Doctor Invalid", AvgConsultationTime: 0, IsOnline: true, CurrentSession: nil},
			},
			position:    3,
			wantMinutes: 0,
			wantErr:     ErrInvalidConsultationTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateEstimatedWaitingTime(tt.doctors, tt.position)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want error %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantMinutes {
				t.Errorf("got %d minutes, want %d minutes", got, tt.wantMinutes)
			}
		})
	}
}
