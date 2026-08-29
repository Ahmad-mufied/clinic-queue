package outbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// AnalyticsRepositoryPort defines the driven/outbound SPI interface for aggregating clinic analytics and doctor performance.
type AnalyticsRepositoryPort interface {
	// GetClinicDailyKPIs computes daily aggregate totals for tickets and wait times.
	GetClinicDailyKPIs(ctx context.Context) (*domain.AnalyticsSummary, error)

	// GetDoctorProductivityList computes performance and utilization metrics for all doctors today.
	GetDoctorProductivityList(ctx context.Context) ([]domain.DoctorPerformance, error)
}
