package inbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// UpdateDoctorConfigDTO defines the input payload for updating doctor consultation configurations.
type UpdateDoctorConfigDTO struct {
	DoctorID               int `json:"doctor_id"`
	AvgConsultationTimeMin int `json:"avg_consultation_time_min"`
}

// AdminUseCase defines the driving/inbound port for executive business analytics and doctor configuration management.
type AdminUseCase interface {
	// GetAnalyticsStats aggregates daily clinic KPIs and doctor productivity metrics.
	GetAnalyticsStats(ctx context.Context) (*domain.AdminDashboardStats, error)

	// UpdateDoctorConfig updates doctor consultation configuration, persists changes, and broadcasts events.
	UpdateDoctorConfig(ctx context.Context, dto UpdateDoctorConfigDTO) (*domain.Doctor, error)
}
