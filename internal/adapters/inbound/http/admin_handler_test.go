package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

type mockAdminUseCase struct {
	getAnalyticsStatsFunc  func(ctx context.Context) (*domain.AdminDashboardStats, error)
	updateDoctorConfigFunc func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error)
}

func (m *mockAdminUseCase) GetAnalyticsStats(ctx context.Context) (*domain.AdminDashboardStats, error) {
	if m != nil && m.getAnalyticsStatsFunc != nil {
		return m.getAnalyticsStatsFunc(ctx)
	}
	return nil, nil
}

func (m *mockAdminUseCase) UpdateDoctorConfig(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
	if m != nil && m.updateDoctorConfigFunc != nil {
		return m.updateDoctorConfigFunc(ctx, dto)
	}
	return nil, nil
}

var _ inbound.AdminUseCase = (*mockAdminUseCase)(nil)

func TestAdminHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(uc *mockAdminUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name: "UseCase error returns 500 Internal Server Error",
			mockSetup: func(uc *mockAdminUseCase) {
				uc.getAnalyticsStatsFunc = func(ctx context.Context) (*domain.AdminDashboardStats, error) {
					return nil, errors.New("db query failure")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name: "Success returns 200 OK with analytics dashboard stats",
			mockSetup: func(uc *mockAdminUseCase) {
				uc.getAnalyticsStatsFunc = func(ctx context.Context) (*domain.AdminDashboardStats, error) {
					return &domain.AdminDashboardStats{
						Summary: domain.AnalyticsSummary{
							TotalServedToday:      42,
							CurrentWaiting:        8,
							CurrentInConsultation: 2,
							AvgActualWaitMinutes:  14.2,
							OnlineDoctorsCount:    2,
						},
						DoctorPerformance: []domain.DoctorPerformance{
							{
								DoctorID:                     "01919df4-8e3b-7412-a1f9-90b567c9e101",
								DoctorName:                   "Doctor A",
								TargetAvgMinutes:             3,
								IsOnline:                     true,
								TotalConsultationsToday:      24,
								AvgActualConsultationMinutes: 3.1,
								UtilizationRatePercentage:    62.0,
							},
						},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"total_served_today":42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockAdminUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewAdminHandler(mockUC)
			err := handler.GetStats(c)
			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if !strings.Contains(rec.Body.String(), tt.wantBodySubstr) {
				t.Errorf("expected body to contain %q, got %s", tt.wantBodySubstr, rec.Body.String())
			}
		})
	}
}

func TestAdminHandler_UpdateDoctorConfig(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(uc *mockAdminUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "Invalid JSON payload returns 400",
			body:           `{invalid-json`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid request payload",
		},
		{
			name: "UseCase returns ErrInvalidInput returns 400",
			body: `{"doctor_id":"   ","avg_consultation_time_min":4}`,
			mockSetup: func(uc *mockAdminUseCase) {
				uc.updateDoctorConfigFunc = func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid input parameters",
		},
		{
			name: "UseCase returns ErrInvalidConsultationTime returns 400",
			body: `{"doctor_id":"01919df4-8e3b-7412-a1f9-90b567c9e101","avg_consultation_time_min":0}`,
			mockSetup: func(uc *mockAdminUseCase) {
				uc.updateDoctorConfigFunc = func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
					return nil, domain.ErrInvalidConsultationTime
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Average consultation time must be greater than 0",
		},
		{
			name: "UseCase returns ErrDoctorNotFound returns 404",
			body: `{"doctor_id":"01919df4-8e3b-7412-a1f9-90b567c9e999","avg_consultation_time_min":4}`,
			mockSetup: func(uc *mockAdminUseCase) {
				uc.updateDoctorConfigFunc = func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
					return nil, domain.ErrDoctorNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Doctor not found",
		},
		{
			name: "UseCase returns generic internal error returns 500",
			body: `{"doctor_id":"01919df4-8e3b-7412-a1f9-90b567c9e101","avg_consultation_time_min":4}`,
			mockSetup: func(uc *mockAdminUseCase) {
				uc.updateDoctorConfigFunc = func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name: "Success returns 200 OK with updated doctor",
			body: `{"doctor_id":"01919df4-8e3b-7412-a1f9-90b567c9e101","avg_consultation_time_min":4}`,
			mockSetup: func(uc *mockAdminUseCase) {
				uc.updateDoctorConfigFunc = func(ctx context.Context, dto inbound.UpdateDoctorConfigDTO) (*domain.Doctor, error) {
					return &domain.Doctor{
						ID:                  "01919df4-8e3b-7412-a1f9-90b567c9e101",
						Name:                "Doctor A",
						AvgConsultationTime: 4,
						IsOnline:            true,
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"avg_consultation_time":4`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/doctors", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockAdminUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewAdminHandler(mockUC)
			err := handler.UpdateDoctorConfig(c)
			if err != nil {
				t.Fatalf("handler returned unexpected error: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if !strings.Contains(rec.Body.String(), tt.wantBodySubstr) {
				t.Errorf("expected body to contain %q, got %s", tt.wantBodySubstr, rec.Body.String())
			}
		})
	}
}

func TestAdminHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	handler := NewAdminHandler(&mockAdminUseCase{})

	dummyMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	handler.RegisterRoutes(e, dummyMW, dummyMW)

	routes := e.Routes()
	expectedRoutes := map[string]string{
		"/api/admin/stats":   http.MethodGet,
		"/api/admin/doctors": http.MethodPost,
	}

	for path, method := range expectedRoutes {
		found := false
		for _, r := range routes {
			if r.Path == path && r.Method == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected route %s %s to be registered", method, path)
		}
	}
}

func TestMockAdminUseCaseDefaults(t *testing.T) {
	var mock mockAdminUseCase
	s, err := mock.GetAnalyticsStats(context.Background())
	if s != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	d, err := mock.UpdateDoctorConfig(context.Background(), inbound.UpdateDoctorConfigDTO{})
	if d != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
}
