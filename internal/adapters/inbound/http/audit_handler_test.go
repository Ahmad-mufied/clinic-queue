package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

type mockAuditUseCase struct {
	recordLogFunc    func(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error)
	getAuditLogsFunc func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error)
}

func (m *mockAuditUseCase) RecordLog(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error) {
	if m != nil && m.recordLogFunc != nil {
		return m.recordLogFunc(ctx, dto)
	}
	return nil, nil
}

func (m *mockAuditUseCase) GetAuditLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	if m != nil && m.getAuditLogsFunc != nil {
		return m.getAuditLogsFunc(ctx, filter)
	}
	return nil, nil
}

var _ inbound.AuditUseCase = (*mockAuditUseCase)(nil)

func TestAuditHandler_GetAuditLogs(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name           string
		queryString    string
		mockSetup      func(uc *mockAuditUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "Invalid non-integer page returns 400",
			queryString:    "?page=abc",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid page parameter",
		},
		{
			name:           "Invalid negative page returns 400",
			queryString:    "?page=-1",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid page parameter",
		},
		{
			name:           "Invalid non-integer limit returns 400",
			queryString:    "?limit=invalid",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid limit parameter",
		},
		{
			name:           "Invalid cursor returns 400",
			queryString:    "?cursor=invalid",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid cursor parameter",
		},
		{
			name:           "Invalid user_id returns 400",
			queryString:    "?user_id=abc",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid user_id parameter",
		},
		{
			name:           "Invalid start_date returns 400",
			queryString:    "?start_date=not-a-date",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid start_date parameter",
		},
		{
			name:           "Invalid end_date returns 400",
			queryString:    "?end_date=not-a-date",
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid end_date parameter",
		},
		{
			name:        "Success with advanced filters and date bounds returns 200 OK",
			queryString: "?search=Doctor&from=2026-08-01&to=2026-08-30&user_id=1&order=asc&cursor=10&limit=15",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					if filter.Search != "Doctor" || filter.SortOrder != "asc" || filter.UserID == nil || *filter.UserID != 1 || filter.StartDate == nil || filter.EndDate == nil {
						return nil, errors.New("filter fields mismatch")
					}
					return &domain.PaginatedAuditLogs{
						Limit:        15,
						TotalRecords: 1,
						Logs: []domain.AuditLog{
							{
								ID:        11,
								ActorName: "Doctor A",
								Role:      "doctor",
								Action:    domain.ActionDoctorShiftStarted,
								CreatedAt: now,
							},
						},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"total_records":1`,
		},
		{
			name:        "UseCase returns ErrInvalidInput returns 400",
			queryString: "?page=1&limit=20",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid input parameters",
		},
		{
			name:        "UseCase returns ErrInvalidAction returns 400",
			queryString: "?page=1&limit=20",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, domain.ErrInvalidAction
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid input parameters",
		},
		{
			name:        "UseCase returns ErrInvalidPage returns 400",
			queryString: "?page=1&limit=20",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, domain.ErrInvalidPage
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid input parameters",
		},
		{
			name:        "UseCase returns ErrInvalidLimit returns 400",
			queryString: "?page=1&limit=20",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, domain.ErrInvalidLimit
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid input parameters",
		},
		{
			name:        "UseCase returns internal error returns 500",
			queryString: "?page=1&limit=20",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, errors.New("database failure")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:        "Success with filters and pagination returns 200 OK",
			queryString: "?action=CONSULTATION_FINISHED&role=doctor&page=2&limit=10",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return &domain.PaginatedAuditLogs{
						Page:         filter.Page,
						Limit:        filter.Limit,
						TotalRecords: 154,
						Logs: []domain.AuditLog{
							{
								ID:        154,
								ActorName: "Doctor A",
								Role:      "doctor",
								Action:    domain.ActionConsultationFinished,
								Details: map[string]any{
									"ticket_id":           101,
									"actual_duration_min": 3.2,
								},
								IPAddress: "127.0.0.1",
								CreatedAt: now,
							},
						},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"total_records":154`,
		},
		{
			name:        "Success with default parameters returns 200 OK",
			queryString: "",
			mockSetup: func(uc *mockAuditUseCase) {
				uc.getAuditLogsFunc = func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return &domain.PaginatedAuditLogs{
						Page:         1,
						Limit:        20,
						TotalRecords: 0,
						Logs:         []domain.AuditLog{},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"total_records":0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs"+tt.queryString, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockAuditUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewAuditHandler(mockUC)
			err := handler.GetAuditLogs(c)
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

func TestAuditHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	handler := NewAuditHandler(&mockAuditUseCase{})

	dummyMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	handler.RegisterRoutes(e, dummyMW, dummyMW)

	routes := e.Routes()
	expectedRoutes := map[string]string{
		"/api/admin/audit-logs": http.MethodGet,
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

func TestMockAuditUseCaseDefaults(t *testing.T) {
	var mock mockAuditUseCase
	log, err := mock.RecordLog(context.Background(), inbound.RecordAuditLogDTO{})
	if log != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	res, err := mock.GetAuditLogs(context.Background(), domain.AuditLogFilter{})
	if res != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
}
