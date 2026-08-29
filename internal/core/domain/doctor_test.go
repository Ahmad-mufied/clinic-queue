package domain

import (
	"testing"
	"time"
)

func TestCalculateDurationMinutes(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		finish   time.Time
		expected float64
	}{
		{
			name:     "Negative difference clamped to 0",
			start:    time.Date(2026, 8, 29, 10, 5, 0, 0, time.UTC),
			finish:   time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "Exact 3 minutes",
			start:    time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
			finish:   time.Date(2026, 8, 29, 10, 3, 0, 0, time.UTC),
			expected: 3.0,
		},
		{
			name:     "3 minutes and 12 seconds rounded to 3.2",
			start:    time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
			finish:   time.Date(2026, 8, 29, 10, 3, 12, 0, time.UTC),
			expected: 3.2,
		},
		{
			name:     "Zero duration",
			start:    time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
			finish:   time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDurationMinutes(tt.start, tt.finish)
			if got != tt.expected {
				t.Errorf("CalculateDurationMinutes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConsultationSession_DurationMinutes(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Minute)
	finishedTime := startTime.Add(3*time.Minute + 30*time.Second)

	tests := []struct {
		name     string
		session  *ConsultationSession
		expected float64
		checkMin bool
	}{
		{
			name:     "Nil session returns 0",
			session:  nil,
			expected: 0,
		},
		{
			name: "Finished session returns rounded duration",
			session: &ConsultationSession{
				StartedAt:  startTime,
				FinishedAt: &finishedTime,
			},
			expected: 3.5,
		},
		{
			name: "Unfinished session calculates duration up to now",
			session: &ConsultationSession{
				StartedAt:  time.Now().Add(-2 * time.Minute),
				FinishedAt: nil,
			},
			checkMin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.session == nil {
				got := tt.session.DurationMinutes()
				if got != tt.expected {
					t.Errorf("DurationMinutes() = %v, want %v", got, tt.expected)
				}
				return
			}

			got := tt.session.DurationMinutes()
			if tt.checkMin {
				if got < 1.9 || got > 2.2 {
					t.Errorf("DurationMinutes() = %v, expected approximately 2.0", got)
				}
			} else if got != tt.expected {
				t.Errorf("DurationMinutes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetermineDoctorStatus(t *testing.T) {
	tests := []struct {
		name             string
		isOnline         bool
		hasActiveSession bool
		expected         DoctorStatus
	}{
		{
			name:             "Offline doctor",
			isOnline:         false,
			hasActiveSession: false,
			expected:         DoctorStatusOffline,
		},
		{
			name:             "Offline doctor with ongoing session remains offline for new queue",
			isOnline:         false,
			hasActiveSession: true,
			expected:         DoctorStatusOffline,
		},
		{
			name:             "Online doctor in active consultation",
			isOnline:         true,
			hasActiveSession: true,
			expected:         DoctorStatusInConsultation,
		},
		{
			name:             "Online doctor available with no active session",
			isOnline:         true,
			hasActiveSession: false,
			expected:         DoctorStatusAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineDoctorStatus(tt.isOnline, tt.hasActiveSession)
			if got != tt.expected {
				t.Errorf("DetermineDoctorStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}
