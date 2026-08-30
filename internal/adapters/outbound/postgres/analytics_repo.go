package postgres

import (
	"context"
	"errors"
	"fmt"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalyticsRepo implements the outbound.AnalyticsRepositoryPort interface using PostgreSQL 18.
type AnalyticsRepo struct {
	pool *pgxpool.Pool
}

// NewAnalyticsRepo constructs a new AnalyticsRepo instance.
func NewAnalyticsRepo(pool *pgxpool.Pool) *AnalyticsRepo {
	return &AnalyticsRepo{pool: pool}
}

var _ outbound.AnalyticsRepositoryPort = (*AnalyticsRepo)(nil)

// GetClinicDailyKPIs aggregates daily clinic KPI metrics from queue tickets.
func (r *AnalyticsRepo) GetClinicDailyKPIs(ctx context.Context) (*domain.AnalyticsSummary, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'COMPLETED') AS total_served_today,
			COUNT(*) FILTER (WHERE status = 'WAITING') AS current_waiting,
			COUNT(*) FILTER (WHERE status = 'IN_CONSULTATION') AS current_in_consultation,
			COALESCE(
				ROUND(AVG(EXTRACT(EPOCH FROM (called_at - created_at)) / 60.0)::numeric, 1), 
				0
			) AS avg_actual_wait_minutes
		FROM queue_tickets
		WHERE created_at >= CURRENT_DATE;
	`

	var (
		totalServed           int
		currentWaiting        int
		currentInConsultation int
		avgWaitMinutes        float64
	)

	err := r.pool.QueryRow(ctx, query).Scan(
		&totalServed,
		&currentWaiting,
		&currentInConsultation,
		&avgWaitMinutes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.AnalyticsSummary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query clinic daily kpis: %w", err)
	}

	return &domain.AnalyticsSummary{
		TotalServedToday:      totalServed,
		CurrentWaiting:        currentWaiting,
		CurrentInConsultation: currentInConsultation,
		AvgActualWaitMinutes:  avgWaitMinutes,
	}, nil
}

// GetDoctorProductivityList aggregates individual doctor productivity and utilization rate metrics.
func (r *AnalyticsRepo) GetDoctorProductivityList(ctx context.Context) ([]domain.DoctorPerformance, error) {
	query := `
		SELECT 
			d.id AS doctor_id,
			d.name AS doctor_name,
			COALESCE(u.username, '') AS username,
			d.avg_consultation_time_min AS target_avg_minutes,
			d.is_online,
			COUNT(cs.id) AS total_consultations_today,
			COALESCE(
				ROUND(AVG(EXTRACT(EPOCH FROM (cs.finished_at - cs.started_at)) / 60.0)::numeric, 1),
				0
			) AS avg_actual_consultation_minutes,
			COALESCE(
				ROUND((SUM(EXTRACT(EPOCH FROM (cs.finished_at - cs.started_at)) / 60.0) / 
				NULLIF(EXTRACT(EPOCH FROM (NOW() - MIN(cs.started_at)) / 60.0), 0) * 100)::numeric, 1),
				0
			) AS utilization_rate_percentage
		FROM doctors d
		LEFT JOIN users u ON d.id = u.doctor_id
		LEFT JOIN consultation_sessions cs ON d.id = cs.doctor_id 
			AND cs.started_at >= CURRENT_DATE 
			AND cs.is_active = FALSE
		GROUP BY d.id, d.name, u.username, d.avg_consultation_time_min, d.is_online
		ORDER BY d.id ASC;
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query doctor productivity list: %w", err)
	}
	defer rows.Close()

	var list []domain.DoctorPerformance
	for rows.Next() {
		var (
			doctorID             string
			doctorName           string
			username             string
			targetAvgMinutes     int
			isOnline             bool
			totalConsultations   int
			avgConsultationMin   float64
			utilizationRatePerc  float64
		)

		if err := rows.Scan(
			&doctorID,
			&doctorName,
			&username,
			&targetAvgMinutes,
			&isOnline,
			&totalConsultations,
			&avgConsultationMin,
			&utilizationRatePerc,
		); err != nil {
			return nil, fmt.Errorf("scan doctor productivity row: %w", err)
		}

		list = append(list, domain.DoctorPerformance{
			DoctorID:                     doctorID,
			DoctorName:                   doctorName,
			Username:                     username,
			TargetAvgMinutes:             targetAvgMinutes,
			IsOnline:                     isOnline,
			TotalConsultationsToday:      totalConsultations,
			AvgActualConsultationMinutes: avgConsultationMin,
			UtilizationRatePercentage:    utilizationRatePerc,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error for doctor productivity: %w", err)
	}

	if list == nil {
		list = []domain.DoctorPerformance{}
	}

	return list, nil
}
