package usecase

import (
	"context"
	"fmt"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

// AdminUseCase implements inbound.AdminUseCase for business analytics and doctor configuration management.
type AdminUseCase struct {
	analyticsRepo outbound.AnalyticsRepositoryPort
	doctorRepo    outbound.DoctorRepositoryPort
	eventPub      outbound.EventPublisherPort
}

// NewAdminUseCase constructs a new AdminUseCase instance.
func NewAdminUseCase(
	analyticsRepo outbound.AnalyticsRepositoryPort,
	doctorRepo outbound.DoctorRepositoryPort,
	eventPub outbound.EventPublisherPort,
) *AdminUseCase {
	return &AdminUseCase{
		analyticsRepo: analyticsRepo,
		doctorRepo:    doctorRepo,
		eventPub:      eventPub,
	}
}

var _ inbound.AdminUseCase = (*AdminUseCase)(nil)

// GetAnalyticsStats aggregates daily clinic KPIs and doctor productivity metrics.
func (u *AdminUseCase) GetAnalyticsStats(ctx context.Context) (*domain.AdminDashboardStats, error) {
	summary, err := u.analyticsRepo.GetClinicDailyKPIs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get clinic daily kpis: %w", err)
	}

	docList, err := u.analyticsRepo.GetDoctorProductivityList(ctx)
	if err != nil {
		return nil, fmt.Errorf("get doctor productivity list: %w", err)
	}

	hourlyFlow, err := u.analyticsRepo.GetHourlyPatientFlow(ctx)
	if err != nil {
		return nil, fmt.Errorf("get hourly patient flow: %w", err)
	}

	if summary == nil {
		summary = &domain.AnalyticsSummary{}
	}
	if docList == nil {
		docList = []domain.DoctorPerformance{}
	}
	if hourlyFlow == nil {
		hourlyFlow = []domain.HourlyPatientFlow{}
	}

	summary.OnlineDoctorsCount = domain.CountOnlineDoctors(docList)

	return &domain.AdminDashboardStats{
		Summary:            *summary,
		DoctorPerformance:  docList,
		HourlyDistribution: hourlyFlow,
	}, nil
}

// UpdateDoctorConfig updates doctor consultation configuration, persists changes, and broadcasts events.
func (u *AdminUseCase) UpdateDoctorConfig(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
	req := &domain.UpdateDoctorConfigRequest{
		DoctorID:               dto.DoctorID,
		AvgConsultationTimeMin: dto.AvgConsultationTimeMin,
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	doc, err := u.doctorRepo.GetDoctorByID(ctx, dto.DoctorID)
	if err != nil {
		return nil, fmt.Errorf("get doctor by id: %w", err)
	}
	if doc == nil {
		return nil, domain.ErrDoctorNotFound
	}

	if err := u.doctorRepo.UpdateDoctorAvgTime(ctx, dto.DoctorID, dto.AvgConsultationTimeMin); err != nil {
		return nil, fmt.Errorf("update doctor avg time: %w", err)
	}

	doc.AvgConsultationTime = dto.AvgConsultationTimeMin

	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "DOCTOR_CONFIG_UPDATED", doc)
		_ = u.eventPub.PublishEvent(ctx, "QUEUE_UPDATED", map[string]any{
			"action":    "DOCTOR_CONFIG_UPDATED",
			"doctor_id": dto.DoctorID,
			"avg_time":  dto.AvgConsultationTimeMin,
		})
	}

	return doc, nil
}
