package domain

import (
	"math"
	"strings"
)

// AnalyticsSummary represents the high-level daily KPI summary for the clinic.
type AnalyticsSummary struct {
	TotalServedToday      int     `json:"total_served_today"`
	CurrentWaiting        int     `json:"current_waiting"`
	CurrentInConsultation int     `json:"current_in_consultation"`
	AvgActualWaitMinutes  float64 `json:"avg_actual_wait_minutes"`
	OnlineDoctorsCount    int     `json:"online_doctors_count"`
}

// DoctorPerformance represents individual doctor productivity, duration metrics, and utilization rates.
type DoctorPerformance struct {
	DoctorID                     string  `json:"doctor_id"`
	DoctorName                   string  `json:"doctor_name"`
	Username                     string  `json:"username,omitempty"`
	TargetAvgMinutes             int     `json:"target_avg_minutes"`
	IsOnline                     bool    `json:"is_online"`
	TotalConsultationsToday      int     `json:"total_consultations_today"`
	AvgActualConsultationMinutes float64 `json:"avg_actual_consultation_minutes"`
	UtilizationRatePercentage    float64 `json:"utilization_rate_percentage"`
}

// HourlyPatientFlow represents time-bucket patient intake counts and peak hour flags.
type HourlyPatientFlow struct {
	HourLabel        string `json:"hour_label"`
	PatientCount     int    `json:"patient_count"`
	HeightPercentage int    `json:"height_percentage"`
	IsPeak           bool   `json:"is_peak"`
}

// AdminDashboardStats aggregates the daily summary cards, doctor productivity table, and hourly flow chart.
type AdminDashboardStats struct {
	Summary            AnalyticsSummary    `json:"summary"`
	DoctorPerformance  []DoctorPerformance `json:"doctor_performance"`
	HourlyDistribution []HourlyPatientFlow `json:"hourly_distribution,omitempty"`
}

// UpdateDoctorConfigRequest defines the domain request parameters to change doctor consultation target time.
type UpdateDoctorConfigRequest struct {
	DoctorID               string `json:"doctor_id"`
	AvgConsultationTimeMin int    `json:"avg_consultation_time_min"`
}

// Validate checks whether the update doctor config request parameters are valid.
func (req *UpdateDoctorConfigRequest) Validate() error {
	if req == nil || strings.TrimSpace(req.DoctorID) == "" {
		return ErrInvalidInput
	}
	if req.AvgConsultationTimeMin <= 0 {
		return ErrInvalidConsultationTime
	}
	return nil
}

// CalculateUtilizationRate computes the utilization percentage rounded to 1 decimal place.
// Returns 0.0 if total shift minutes is non-positive or consultation minutes is non-positive.
func CalculateUtilizationRate(totalConsultationMinutes, totalShiftMinutes float64) float64 {
	if totalShiftMinutes <= 0 || totalConsultationMinutes <= 0 {
		return 0.0
	}
	rate := (totalConsultationMinutes / totalShiftMinutes) * 100.0
	return math.Round(rate*10) / 10
}

// CalculateDurationDelta computes the variance between actual consultation duration and configured target average.
// Rounded to 1 decimal place.
func CalculateDurationDelta(actualAvgMinutes float64, targetAvgMinutes int) float64 {
	delta := actualAvgMinutes - float64(targetAvgMinutes)
	return math.Round(delta*10) / 10
}

// CountOnlineDoctors calculates the count of currently online doctors from a performance slice.
func CountOnlineDoctors(doctors []DoctorPerformance) int {
	count := 0
	for _, doc := range doctors {
		if doc.IsOnline {
			count++
		}
	}
	return count
}
