package usecase

import (
	"context"
	"errors"
	"testing"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

type mockAnalyticsRepoPort struct {
	getClinicDailyKPIsFunc        func(ctx context.Context) (*domain.AnalyticsSummary, error)
	getDoctorProductivityListFunc func(ctx context.Context) ([]domain.DoctorPerformance, error)
}

func (m *mockAnalyticsRepoPort) GetClinicDailyKPIs(ctx context.Context) (*domain.AnalyticsSummary, error) {
	if m != nil && m.getClinicDailyKPIsFunc != nil {
		return m.getClinicDailyKPIsFunc(ctx)
	}
	return nil, nil
}

func (m *mockAnalyticsRepoPort) GetDoctorProductivityList(ctx context.Context) ([]domain.DoctorPerformance, error) {
	if m != nil && m.getDoctorProductivityListFunc != nil {
		return m.getDoctorProductivityListFunc(ctx)
	}
	return nil, nil
}

type mockAdminDoctorRepoPort struct {
	getActiveDoctorsFunc          func(ctx context.Context) ([]*domain.Doctor, error)
	getAllDoctorsWithSessionsFunc func(ctx context.Context) ([]domain.DoctorAvailability, error)
	getDoctorByIDFunc             func(ctx context.Context, id string) (*domain.Doctor, error)
	updateOnlineStatusFunc        func(ctx context.Context, doctorID string, isOnline bool) error
	getActiveSessionByDoctorID    func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error)
	updateDoctorAvgTimeFunc       func(ctx context.Context, doctorID string, avgTime int) error
}

func (m *mockAdminDoctorRepoPort) GetActiveDoctors(ctx context.Context) ([]*domain.Doctor, error) {
	if m != nil && m.getActiveDoctorsFunc != nil {
		return m.getActiveDoctorsFunc(ctx)
	}
	return nil, nil
}

func (m *mockAdminDoctorRepoPort) GetAllDoctorsWithSessions(ctx context.Context) ([]domain.DoctorAvailability, error) {
	if m != nil && m.getAllDoctorsWithSessionsFunc != nil {
		return m.getAllDoctorsWithSessionsFunc(ctx)
	}
	return nil, nil
}

func (m *mockAdminDoctorRepoPort) GetDoctorByID(ctx context.Context, id string) (*domain.Doctor, error) {
	if m != nil && m.getDoctorByIDFunc != nil {
		return m.getDoctorByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAdminDoctorRepoPort) UpdateOnlineStatus(ctx context.Context, doctorID string, isOnline bool) error {
	if m != nil && m.updateOnlineStatusFunc != nil {
		return m.updateOnlineStatusFunc(ctx, doctorID, isOnline)
	}
	return nil
}

func (m *mockAdminDoctorRepoPort) GetActiveSessionByDoctorID(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	if m != nil && m.getActiveSessionByDoctorID != nil {
		return m.getActiveSessionByDoctorID(ctx, doctorID)
	}
	return nil, nil
}

func (m *mockAdminDoctorRepoPort) UpdateDoctorAvgTime(ctx context.Context, doctorID string, avgTime int) error {
	if m != nil && m.updateDoctorAvgTimeFunc != nil {
		return m.updateDoctorAvgTimeFunc(ctx, doctorID, avgTime)
	}
	return nil
}

type mockAdminEventPubPort struct {
	publishEventFunc func(ctx context.Context, eventType string, payload any) error
	closeFunc        func() error
}

func (m *mockAdminEventPubPort) PublishEvent(ctx context.Context, eventType string, payload any) error {
	if m != nil && m.publishEventFunc != nil {
		return m.publishEventFunc(ctx, eventType, payload)
	}
	return nil
}

func (m *mockAdminEventPubPort) Close() error {
	if m != nil && m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

var (
	_ outbound.AnalyticsRepositoryPort = (*mockAnalyticsRepoPort)(nil)
	_ outbound.DoctorRepositoryPort    = (*mockAdminDoctorRepoPort)(nil)
	_ outbound.EventPublisherPort      = (*mockAdminEventPubPort)(nil)
)

func TestAdminUseCase_GetAnalyticsStats(t *testing.T) {
	tests := []struct {
		name          string
		analyticsRepo *mockAnalyticsRepoPort
		doctorRepo    *mockAdminDoctorRepoPort
		wantErr       error
		wantServed    int
		wantOnline    int
		wantDocCount  int
	}{
		{
			name: "GetClinicDailyKPIs error",
			analyticsRepo: &mockAnalyticsRepoPort{
				getClinicDailyKPIsFunc: func(ctx context.Context) (*domain.AnalyticsSummary, error) {
					return nil, errors.New("kpi query failed")
				},
			},
			wantErr: errors.New("kpi query failed"),
		},
		{
			name: "GetDoctorProductivityList error",
			analyticsRepo: &mockAnalyticsRepoPort{
				getClinicDailyKPIsFunc: func(ctx context.Context) (*domain.AnalyticsSummary, error) {
					return &domain.AnalyticsSummary{
						TotalServedToday: 42,
					}, nil
				},
				getDoctorProductivityListFunc: func(ctx context.Context) ([]domain.DoctorPerformance, error) {
					return nil, errors.New("doctor productivity query failed")
				},
			},
			wantErr: errors.New("doctor productivity query failed"),
		},
		{
			name: "Success with populated summary and doctor productivity list",
			analyticsRepo: &mockAnalyticsRepoPort{
				getClinicDailyKPIsFunc: func(ctx context.Context) (*domain.AnalyticsSummary, error) {
					return &domain.AnalyticsSummary{
						TotalServedToday:      42,
						CurrentWaiting:        8,
						CurrentInConsultation: 2,
						AvgActualWaitMinutes:  14.2,
					}, nil
				},
				getDoctorProductivityListFunc: func(ctx context.Context) ([]domain.DoctorPerformance, error) {
					return []domain.DoctorPerformance{
						{
							DoctorID:                     "01919df4-8e3b-7412-a1f9-90b567c9e101",
							DoctorName:                   "Doctor A",
							TargetAvgMinutes:             3,
							IsOnline:                     true,
							TotalConsultationsToday:      24,
							AvgActualConsultationMinutes: 3.1,
							UtilizationRatePercentage:    62.0,
						},
						{
							DoctorID:                     "01919df4-8e3b-7412-a1f9-90b567c9e102",
							DoctorName:                   "Doctor B",
							TargetAvgMinutes:             4,
							IsOnline:                     false,
							TotalConsultationsToday:      18,
							AvgActualConsultationMinutes: 3.9,
							UtilizationRatePercentage:    58.5,
						},
					}, nil
				},
			},
			wantErr:      nil,
			wantServed:   42,
			wantOnline:   1,
			wantDocCount: 2,
		},
		{
			name: "Success when summary is nil and docList is nil (Zero State)",
			analyticsRepo: &mockAnalyticsRepoPort{
				getClinicDailyKPIsFunc: func(ctx context.Context) (*domain.AnalyticsSummary, error) {
					return nil, nil
				},
				getDoctorProductivityListFunc: func(ctx context.Context) ([]domain.DoctorPerformance, error) {
					return nil, nil
				},
			},
			wantErr:      nil,
			wantServed:   0,
			wantOnline:   0,
			wantDocCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAdminUseCase(tt.analyticsRepo, tt.doctorRepo, nil)
			stats, err := uc.GetAnalyticsStats(context.Background())

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error message %q, got %q", tt.wantErr.Error(), err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if stats.Summary.TotalServedToday != tt.wantServed {
				t.Errorf("expected TotalServedToday %d, got %d", tt.wantServed, stats.Summary.TotalServedToday)
			}
			if stats.Summary.OnlineDoctorsCount != tt.wantOnline {
				t.Errorf("expected OnlineDoctorsCount %d, got %d", tt.wantOnline, stats.Summary.OnlineDoctorsCount)
			}
			if len(stats.DoctorPerformance) != tt.wantDocCount {
				t.Errorf("expected DoctorPerformance count %d, got %d", tt.wantDocCount, len(stats.DoctorPerformance))
			}
		})
	}
}

func TestAdminUseCase_UpdateDoctorConfig(t *testing.T) {
	tests := []struct {
		name          string
		dto           inbound.UpdateDoctorConfigDTO
		doctorRepo    *mockAdminDoctorRepoPort
		eventPub      *mockAdminEventPubPort
		wantErr       error
		wantAvgTime   int
		wantPubEvents int
	}{
		{
			name: "Invalid doctor ID empty returns ErrInvalidInput",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "",
				AvgConsultationTimeMin: 4,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "Invalid doctor ID whitespace returns ErrInvalidInput",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "   ",
				AvgConsultationTimeMin: 4,
			},
			wantErr: domain.ErrInvalidInput,
		},
		{
			name: "Invalid avg consultation time <= 0 returns ErrInvalidConsultationTime",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e101",
				AvgConsultationTimeMin: 0,
			},
			wantErr: domain.ErrInvalidConsultationTime,
		},
		{
			name: "GetDoctorByID error",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e101",
				AvgConsultationTimeMin: 4,
			},
			doctorRepo: &mockAdminDoctorRepoPort{
				getDoctorByIDFunc: func(ctx context.Context, id string) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "Doctor not found returns ErrDoctorNotFound",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e999",
				AvgConsultationTimeMin: 4,
			},
			doctorRepo: &mockAdminDoctorRepoPort{
				getDoctorByIDFunc: func(ctx context.Context, id string) (*domain.Doctor, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrDoctorNotFound,
		},
		{
			name: "UpdateDoctorAvgTime error",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e101",
				AvgConsultationTimeMin: 4,
			},
			doctorRepo: &mockAdminDoctorRepoPort{
				getDoctorByIDFunc: func(ctx context.Context, id string) (*domain.Doctor, error) {
					return &domain.Doctor{
						ID:                  "01919df4-8e3b-7412-a1f9-90b567c9e101",
						Name:                "Doctor A",
						AvgConsultationTime: 3,
					}, nil
				},
				updateDoctorAvgTimeFunc: func(ctx context.Context, doctorID string, avgTime int) error {
					return errors.New("update query failed")
				},
			},
			wantErr: errors.New("update query failed"),
		},
		{
			name: "Success with NATS event broadcasting",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e101",
				AvgConsultationTimeMin: 5,
			},
			doctorRepo: &mockAdminDoctorRepoPort{
				getDoctorByIDFunc: func(ctx context.Context, id string) (*domain.Doctor, error) {
					return &domain.Doctor{
						ID:                  "01919df4-8e3b-7412-a1f9-90b567c9e101",
						Name:                "Doctor A",
						AvgConsultationTime: 3,
					}, nil
				},
				updateDoctorAvgTimeFunc: func(ctx context.Context, doctorID string, avgTime int) error {
					return nil
				},
			},
			eventPub: &mockAdminEventPubPort{
				publishEventFunc: func(ctx context.Context, eventType string, payload any) error {
					return nil
				},
			},
			wantErr:     nil,
			wantAvgTime: 5,
		},
		{
			name: "Success with nil eventPub",
			dto: inbound.UpdateDoctorConfigDTO{
				DoctorID:               "01919df4-8e3b-7412-a1f9-90b567c9e101",
				AvgConsultationTimeMin: 4,
			},
			doctorRepo: &mockAdminDoctorRepoPort{
				getDoctorByIDFunc: func(ctx context.Context, id string) (*domain.Doctor, error) {
					return &domain.Doctor{
						ID:                  "01919df4-8e3b-7412-a1f9-90b567c9e101",
						Name:                "Doctor A",
						AvgConsultationTime: 3,
					}, nil
				},
				updateDoctorAvgTimeFunc: func(ctx context.Context, doctorID string, avgTime int) error {
					return nil
				},
			},
			eventPub:    nil,
			wantErr:     nil,
			wantAvgTime: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAdminUseCase(nil, tt.doctorRepo, tt.eventPub)
			doc, err := uc.UpdateDoctorConfig(context.Background(), tt.dto)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if doc.AvgConsultationTime != tt.wantAvgTime {
				t.Errorf("expected AvgConsultationTime %d, got %d", tt.wantAvgTime, doc.AvgConsultationTime)
			}
		})
	}
}

func TestMockDefaults(t *testing.T) {
	// Call default branches of mocks when functions are nil to ensure 100% coverage
	var analyticsMock mockAnalyticsRepoPort
	s, err := analyticsMock.GetClinicDailyKPIs(context.Background())
	if s != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	l, err := analyticsMock.GetDoctorProductivityList(context.Background())
	if l != nil || err != nil {
		t.Errorf("expected nil, nil")
	}

	var docMock mockAdminDoctorRepoPort
	dList, err := docMock.GetActiveDoctors(context.Background())
	if dList != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	availList, err := docMock.GetAllDoctorsWithSessions(context.Background())
	if availList != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	d, err := docMock.GetDoctorByID(context.Background(), "01919df4-8e3b-7412-a1f9-90b567c9e101")
	if d != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	err = docMock.UpdateOnlineStatus(context.Background(), "01919df4-8e3b-7412-a1f9-90b567c9e101", true)
	if err != nil {
		t.Errorf("expected nil")
	}
	sess, err := docMock.GetActiveSessionByDoctorID(context.Background(), "01919df4-8e3b-7412-a1f9-90b567c9e101")
	if sess != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	err = docMock.UpdateDoctorAvgTime(context.Background(), "01919df4-8e3b-7412-a1f9-90b567c9e101", 3)
	if err != nil {
		t.Errorf("expected nil")
	}

	var eventMock mockAdminEventPubPort
	err = eventMock.PublishEvent(context.Background(), "EVT", nil)
	if err != nil {
		t.Errorf("expected nil")
	}
	err = eventMock.Close()
	if err != nil {
		t.Errorf("expected nil")
	}
}
