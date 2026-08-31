package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"

	"github.com/labstack/echo/v4"
)

type mockQueueUseCase struct {
	joinQueueFunc      func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error)
	getMyTicketFunc    func(ctx context.Context, userID string) (*domain.QueueTicket, error)
	getQueueStatusFunc func(ctx context.Context) (*domain.QueueStatus, error)
}

func (m *mockQueueUseCase) JoinQueue(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
	if m.joinQueueFunc != nil {
		return m.joinQueueFunc(ctx, userID, patientName)
	}
	return nil, nil
}

func (m *mockQueueUseCase) GetMyTicket(ctx context.Context, userID string) (*domain.QueueTicket, error) {
	if m.getMyTicketFunc != nil {
		return m.getMyTicketFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockQueueUseCase) GetQueueStatus(ctx context.Context) (*domain.QueueStatus, error) {
	if m.getQueueStatusFunc != nil {
		return m.getQueueStatusFunc(ctx)
	}
	return nil, nil
}

func TestQueueHandler_JoinQueue(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setContextUser *string
		mockSetup      func(uc *mockQueueUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "Invalid JSON payload returns 400",
			body:           `{invalid json`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid request payload",
		},
		{
			name:           "Empty patient name in body returns 400",
			body:           `{"patient_name": "   "}`,
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Patient name is required",
		},
		{
			name:           "UseCase returns ErrInvalidInput -> 400",
			body:           `{"patient_name": "John"}`,
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Patient name is required",
		},
		{
			name:           "UseCase returns ErrActiveTicketExists -> 409 Conflict",
			body:           `{"patient_name": "John"}`,
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					return nil, domain.ErrActiveTicketExists
				}
			},
			wantStatus:     http.StatusConflict,
			wantBodySubstr: "Active queue ticket already exists",
		},
		{
			name:           "UseCase returns ErrNoDoctorsAvailable -> 503 Service Unavailable",
			body:           `{"patient_name": "John"}`,
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					return nil, domain.ErrNoDoctorsAvailable
				}
			},
			wantStatus:     http.StatusServiceUnavailable,
			wantBodySubstr: "No doctors currently configured for this clinic",
		},
		{
			name:           "UseCase returns internal error -> 500",
			body:           `{"patient_name": "John"}`,
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					return nil, errors.New("database unreachable")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:           "Success: Authenticated patient joins queue -> 201 Created",
			body:           `{"patient_name": "John Doe"}`,
			setContextUser: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e201"),
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					wait := 16
					return &domain.QueueTicket{
						ID:                       "01919df4-8e3b-7412-a1f9-90b567c9e401",
						UserID:                   userID,
						PatientName:              patientName,
						QueueNumber:              "A-11",
						Status:                   domain.TicketStatusWaiting,
						PositionInQueue:          11,
						AheadCount:               10,
						EstimatedWaitTimeMinutes: &wait,
						CreatedAt:                time.Now(),
					}, nil
				}
			},
			wantStatus:     http.StatusCreated,
			wantBodySubstr: `"queue_number":"A-11"`,
		},
		{
			name:           "Success: Walk-in guest joins without context user -> 201 Created",
			body:           `{"patient_name": "Alice Walker"}`,
			setContextUser: nil,
			mockSetup: func(uc *mockQueueUseCase) {
				uc.joinQueueFunc = func(ctx context.Context, userID *string, patientName string) (*domain.QueueTicket, error) {
					wait := 0
					return &domain.QueueTicket{
						ID:                       "01919df4-8e3b-7412-a1f9-90b567c9e402",
						PatientName:              patientName,
						QueueNumber:              "A-01",
						Status:                   domain.TicketStatusWaiting,
						PositionInQueue:          1,
						AheadCount:               0,
						EstimatedWaitTimeMinutes: &wait,
						CreatedAt:                time.Now(),
					}, nil
				}
			},
			wantStatus:     http.StatusCreated,
			wantBodySubstr: `"queue_number":"A-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/queue/join", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextUser != nil {
				c.Set(middleware.ContextKeyUserID, *tt.setContextUser)
			}

			mockUC := &mockQueueUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewQueueHandler(mockUC)
			err := handler.JoinQueue(c)
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

func TestQueueHandler_GetMyTicket(t *testing.T) {
	tests := []struct {
		name           string
		setContextUser *string
		mockSetup      func(uc *mockQueueUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name:           "Unauthenticated request without user ID in context returns 401",
			setContextUser: nil,
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "Unauthorized",
		},
		{
			name:           "Context user ID empty returns 401",
			setContextUser: strPtr(""),
			wantStatus:     http.StatusUnauthorized,
			wantBodySubstr: "Unauthorized",
		},
		{
			name:           "Ticket not found returns 404",
			setContextUser: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e201"),
			mockSetup: func(uc *mockQueueUseCase) {
				uc.getMyTicketFunc = func(ctx context.Context, userID string) (*domain.QueueTicket, error) {
					return nil, domain.ErrTicketNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "No active ticket found",
		},
		{
			name:           "UseCase returns ErrInvalidInput -> 400",
			setContextUser: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e201"),
			mockSetup: func(uc *mockQueueUseCase) {
				uc.getMyTicketFunc = func(ctx context.Context, userID string) (*domain.QueueTicket, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Patient name is required",
		},
		{
			name:           "UseCase returns internal error -> 500",
			setContextUser: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e201"),
			mockSetup: func(uc *mockQueueUseCase) {
				uc.getMyTicketFunc = func(ctx context.Context, userID string) (*domain.QueueTicket, error) {
					return nil, errors.New("query failure")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:           "Success: Returns active ticket with wait time -> 200 OK",
			setContextUser: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e201"),
			mockSetup: func(uc *mockQueueUseCase) {
				uc.getMyTicketFunc = func(ctx context.Context, userID string) (*domain.QueueTicket, error) {
					wait := 12
					return &domain.QueueTicket{
						ID:                       "01919df4-8e3b-7412-a1f9-90b567c9e401",
						UserID:                   &userID,
						PatientName:              "John Doe",
						QueueNumber:              "A-05",
						Status:                   domain.TicketStatusWaiting,
						PositionInQueue:          5,
						AheadCount:               4,
						EstimatedWaitTimeMinutes: &wait,
						CreatedAt:                time.Now(),
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"queue_number":"A-05"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/queue/my-ticket", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextUser != nil {
				c.Set(middleware.ContextKeyUserID, *tt.setContextUser)
			}

			mockUC := &mockQueueUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewQueueHandler(mockUC)
			err := handler.GetMyTicket(c)
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

func TestQueueHandler_GetQueueStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(uc *mockQueueUseCase)
		wantStatus     int
		wantBodySubstr string
	}{
		{
			name: "UseCase error returns 500",
			mockSetup: func(uc *mockQueueUseCase) {
				uc.getQueueStatusFunc = func(ctx context.Context) (*domain.QueueStatus, error) {
					return nil, errors.New("database down")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name: "Success returns 200 OK with queue status",
			mockSetup: func(uc *mockQueueUseCase) {
				wait := 0
				uc.getQueueStatusFunc = func(ctx context.Context) (*domain.QueueStatus, error) {
					return &domain.QueueStatus{
						OnlineDoctors: []domain.DoctorAvailability{
							{ID: "01919df4-8e3b-7412-a1f9-90b567c9e101", Name: "Doctor A", AvgConsultationTimeMinutes: 3, IsOnline: true, Status: domain.DoctorStatusAvailable},
						},
						TotalWaiting: 1,
						QueueList: []domain.QueueTicketSummary{
							{QueueNumber: "A-01", PatientName: "Alice", EstimatedWaitMinutes: &wait},
						},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"total_waiting":1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/queue/status", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockQueueUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewQueueHandler(mockUC)
			err := handler.GetQueueStatus(c)
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

func TestQueueHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	handler := NewQueueHandler(&mockQueueUseCase{})

	dummyMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
	dummyRateMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	handler.RegisterRoutes(e, dummyMW, dummyMW, dummyRateMW)

	routes := e.Routes()
	expectedRoutes := map[string]string{
		"/api/queue/join":      http.MethodPost,
		"/api/queue/my-ticket": http.MethodGet,
		"/api/queue/status":    http.MethodGet,
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

	// Test without rate limiter
	e2 := echo.New()
	handler.RegisterRoutes(e2, dummyMW, dummyMW)
	if len(e2.Routes()) == 0 {
		t.Error("expected routes registered on e2")
	}
}

