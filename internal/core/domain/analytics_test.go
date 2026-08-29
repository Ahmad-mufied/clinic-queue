package domain

import (
	"errors"
	"testing"
)

func TestCalculateUtilizationRate(t *testing.T) {
	tests := []struct {
		name                 string
		consultationMinutes  float64
		shiftMinutes         float64
		wantUtilizationRate float64
	}{
		{
			name:                 "Zero shift minutes returns 0",
			consultationMinutes:  50.0,
			shiftMinutes:         0.0,
			wantUtilizationRate: 0.0,
		},
		{
			name:                 "Negative shift minutes returns 0",
			consultationMinutes:  50.0,
			shiftMinutes:         -10.0,
			wantUtilizationRate: 0.0,
		},
		{
			name:                 "Zero consultation minutes returns 0",
			consultationMinutes:  0.0,
			shiftMinutes:         120.0,
			wantUtilizationRate: 0.0,
		},
		{
			name:                 "Negative consultation minutes returns 0",
			consultationMinutes:  -10.0,
			shiftMinutes:         120.0,
			wantUtilizationRate: 0.0,
		},
		{
			name:                 "Valid calculation with rounding (72m consultation in 240m shift = 30.0%)",
			consultationMinutes:  72.0,
			shiftMinutes:         240.0,
			wantUtilizationRate: 30.0,
		},
		{
			name:                 "Valid calculation with 1 decimal rounding (70m consultation in 240m shift = 29.166... -> 29.2%)",
			consultationMinutes:  70.0,
			shiftMinutes:         240.0,
			wantUtilizationRate: 29.2,
		},
		{
			name:                 "100% full utilization",
			consultationMinutes:  100.0,
			shiftMinutes:         100.0,
			wantUtilizationRate: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateUtilizationRate(tt.consultationMinutes, tt.shiftMinutes)
			if got != tt.wantUtilizationRate {
				t.Errorf("CalculateUtilizationRate(%v, %v) = %v; want %v",
					tt.consultationMinutes, tt.shiftMinutes, got, tt.wantUtilizationRate)
			}
		})
	}
}

func TestCalculateDurationDelta(t *testing.T) {
	tests := []struct {
		name             string
		actualAvgMinutes float64
		targetAvgMinutes int
		wantDelta        float64
	}{
		{
			name:             "Positive delta (actual > target)",
			actualAvgMinutes: 3.8,
			targetAvgMinutes: 3,
			wantDelta:        0.8,
		},
		{
			name:             "Negative delta (actual < target)",
			actualAvgMinutes: 2.6,
			targetAvgMinutes: 3,
			wantDelta:        -0.4,
		},
		{
			name:             "Zero delta (actual == target)",
			actualAvgMinutes: 4.0,
			targetAvgMinutes: 4,
			wantDelta:        0.0,
		},
		{
			name:             "Decimal delta rounding check",
			actualAvgMinutes: 3.123,
			targetAvgMinutes: 3,
			wantDelta:        0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDurationDelta(tt.actualAvgMinutes, tt.targetAvgMinutes)
			if got != tt.wantDelta {
				t.Errorf("CalculateDurationDelta(%v, %v) = %v; want %v",
					tt.actualAvgMinutes, tt.targetAvgMinutes, got, tt.wantDelta)
			}
		})
	}
}

func TestCountOnlineDoctors(t *testing.T) {
	tests := []struct {
		name      string
		doctors   []DoctorPerformance
		wantCount int
	}{
		{
			name:      "Empty doctor list returns 0",
			doctors:   nil,
			wantCount: 0,
		},
		{
			name: "All doctors online",
			doctors: []DoctorPerformance{
				{DoctorID: 1, IsOnline: true},
				{DoctorID: 2, IsOnline: true},
			},
			wantCount: 2,
		},
		{
			name: "All doctors offline",
			doctors: []DoctorPerformance{
				{DoctorID: 1, IsOnline: false},
				{DoctorID: 2, IsOnline: false},
			},
			wantCount: 0,
		},
		{
			name: "Mixed online and offline doctors",
			doctors: []DoctorPerformance{
				{DoctorID: 1, IsOnline: true},
				{DoctorID: 2, IsOnline: false},
				{DoctorID: 3, IsOnline: true},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountOnlineDoctors(tt.doctors)
			if got != tt.wantCount {
				t.Errorf("CountOnlineDoctors() = %d; want %d", got, tt.wantCount)
			}
		})
	}
}

func TestUpdateDoctorConfigRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *UpdateDoctorConfigRequest
		wantErr error
	}{
		{
			name:    "Nil request returns ErrInvalidInput",
			req:     nil,
			wantErr: ErrInvalidInput,
		},
		{
			name: "Doctor ID zero returns ErrInvalidInput",
			req: &UpdateDoctorConfigRequest{
				DoctorID:               0,
				AvgConsultationTimeMin: 3,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Doctor ID negative returns ErrInvalidInput",
			req: &UpdateDoctorConfigRequest{
				DoctorID:               -1,
				AvgConsultationTimeMin: 3,
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Consultation time zero returns ErrInvalidConsultationTime",
			req: &UpdateDoctorConfigRequest{
				DoctorID:               1,
				AvgConsultationTimeMin: 0,
			},
			wantErr: ErrInvalidConsultationTime,
		},
		{
			name: "Consultation time negative returns ErrInvalidConsultationTime",
			req: &UpdateDoctorConfigRequest{
				DoctorID:               1,
				AvgConsultationTimeMin: -5,
			},
			wantErr: ErrInvalidConsultationTime,
		},
		{
			name: "Valid request returns nil error",
			req: &UpdateDoctorConfigRequest{
				DoctorID:               1,
				AvgConsultationTimeMin: 4,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v; want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}
