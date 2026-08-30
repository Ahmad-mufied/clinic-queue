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

type mockDoctorUseCase struct {
	toggleStatusFunc       func(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error)
	callNextPatientFunc    func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error)
	finishConsultationFunc func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error)
	getWorkspaceFunc       func(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error)
}

func (m *mockDoctorUseCase) ToggleStatus(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
	if m.toggleStatusFunc != nil {
		return m.toggleStatusFunc(ctx, doctorID, isOnline)
	}
	return nil, nil
}

func (m *mockDoctorUseCase) CallNextPatient(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
	if m.callNextPatientFunc != nil {
		return m.callNextPatientFunc(ctx, doctorID)
	}
	return nil, nil
}

func (m *mockDoctorUseCase) FinishConsultation(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
	if m.finishConsultationFunc != nil {
		return m.finishConsultationFunc(ctx, doctorID)
	}
	return nil, nil
}

func (m *mockDoctorUseCase) GetWorkspace(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
	if m.getWorkspaceFunc != nil {
		return m.getWorkspaceFunc(ctx, doctorID)
	}
	return nil, nil
}

func strPtr(s string) *string {
	return &s
}

func TestDoctorHandler_ToggleStatus(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setContextDoctor *string
		mockSetup        func(uc *mockDoctorUseCase)
		wantStatus       int
		wantBodySubstr   string
	}{
		{
			name:             "Missing doctor ID in context returns 403",
			body:             `{"is_online": true}`,
			setContextDoctor: nil,
			wantStatus:       http.StatusForbidden,
			wantBodySubstr:   "Doctor profile required",
		},
		{
			name:             "Invalid doctor ID empty returns 403",
			body:             `{"is_online": true}`,
			setContextDoctor: strPtr(""),
			wantStatus:       http.StatusForbidden,
			wantBodySubstr:   "Doctor profile required",
		},
		{
			name:             "Invalid JSON payload returns 400",
			body:             `{invalid json`,
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			wantStatus:       http.StatusBadRequest,
			wantBodySubstr:   "Invalid request payload",
		},
		{
			name:             "Doctor not found returns 404",
			body:             `{"is_online": true}`,
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e999"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.toggleStatusFunc = func(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
					return nil, domain.ErrDoctorNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Doctor not found",
		},
		{
			name:             "UseCase returns ErrInvalidInput -> 400",
			body:             `{"is_online": true}`,
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.toggleStatusFunc = func(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid doctor ID",
		},
		{
			name:             "UseCase returns internal error -> 500",
			body:             `{"is_online": true}`,
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.toggleStatusFunc = func(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
					return nil, errors.New("database update error")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:             "Success returns 200 OK with updated shift status",
			body:             `{"is_online": true}`,
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.toggleStatusFunc = func(ctx context.Context, doctorID string, isOnline bool) (*domain.DoctorShiftResponse, error) {
					return &domain.DoctorShiftResponse{
						DoctorID: "01919df4-8e3b-7412-a1f9-90b567c9e101",
						Name:     "Doctor A",
						IsOnline: true,
						Status:   domain.DoctorStatusAvailable,
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"status":"AVAILABLE"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/doctors/status", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextDoctor != nil {
				c.Set(middleware.ContextKeyDoctorID, tt.setContextDoctor)
			}

			mockUC := &mockDoctorUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewDoctorHandler(mockUC)
			err := handler.ToggleStatus(c)
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

func TestDoctorHandler_CallNextPatient(t *testing.T) {
	tests := []struct {
		name             string
		setContextDoctor *string
		mockSetup        func(uc *mockDoctorUseCase)
		wantStatus       int
		wantBodySubstr   string
	}{
		{
			name:             "Missing doctor ID in context returns 403",
			setContextDoctor: nil,
			wantStatus:       http.StatusForbidden,
			wantBodySubstr:   "Doctor profile required",
		},
		{
			name:             "Doctor is offline returns 400 Bad Request",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, domain.ErrDoctorOffline
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Doctor must be online to call patients",
		},
		{
			name:             "Active consultation exists returns 409 Conflict",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, domain.ErrActiveConsultationExists
				}
			},
			wantStatus:     http.StatusConflict,
			wantBodySubstr: "Active consultation already in progress",
		},
		{
			name:             "Queue is empty returns 200 OK with empty notice",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, domain.ErrQueueEmpty
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: "Queue is empty. No patients waiting.",
		},
		{
			name:             "Doctor not found returns 404",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e999"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, domain.ErrDoctorNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Doctor not found",
		},
		{
			name:             "UseCase returns ErrInvalidInput -> 400",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid doctor ID",
		},
		{
			name:             "UseCase returns generic internal error -> 500",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:             "Success returns 200 OK with session data",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.callNextPatientFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{
						ID:          "01919df4-8e3b-7412-a1f9-90b567c9e301",
						DoctorID:    "01919df4-8e3b-7412-a1f9-90b567c9e101",
						TicketID:    "01919df4-8e3b-7412-a1f9-90b567c9e401",
						PatientName: "Alice",
						StartedAt:   time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
						IsActive:    true,
						Ticket: &domain.ConsultationTicket{
							ID:          "01919df4-8e3b-7412-a1f9-90b567c9e401",
							QueueNumber: "A-01",
							PatientName: "Alice",
							Status:      domain.TicketStatusInConsultation,
						},
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"session_id":"01919df4-8e3b-7412-a1f9-90b567c9e301"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/doctors/call-next", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextDoctor != nil {
				c.Set(middleware.ContextKeyDoctorID, tt.setContextDoctor)
			}

			mockUC := &mockDoctorUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewDoctorHandler(mockUC)
			err := handler.CallNextPatient(c)
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

func TestDoctorHandler_FinishConsultation(t *testing.T) {
	tests := []struct {
		name             string
		setContextDoctor *string
		mockSetup        func(uc *mockDoctorUseCase)
		wantStatus       int
		wantBodySubstr   string
	}{
		{
			name:             "Missing doctor ID in context returns 403",
			setContextDoctor: nil,
			wantStatus:       http.StatusForbidden,
			wantBodySubstr:   "Doctor profile required",
		},
		{
			name:             "No active consultation found returns 400 Bad Request",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.finishConsultationFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
					return nil, domain.ErrNoActiveConsultation
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "No active consultation found to finish",
		},
		{
			name:             "Doctor not found returns 404",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e999"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.finishConsultationFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
					return nil, domain.ErrDoctorNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Doctor not found",
		},
		{
			name:             "UseCase returns ErrInvalidInput -> 400",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.finishConsultationFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid doctor ID",
		},
		{
			name:             "UseCase returns internal error -> 500",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.finishConsultationFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
					return nil, errors.New("update query failed")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:             "Success returns 200 OK with finish consultation response",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.finishConsultationFunc = func(ctx context.Context, doctorID string) (*domain.ConsultationFinishResponse, error) {
					return &domain.ConsultationFinishResponse{
						SessionID:             "01919df4-8e3b-7412-a1f9-90b567c9e301",
						PatientName:           "Alice",
						ActualDurationMinutes: 3.2,
						FinishedAt:            time.Date(2026, 8, 29, 10, 3, 12, 0, time.UTC),
						DoctorStatus:          domain.DoctorStatusAvailable,
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"actual_duration_minutes":3.2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/doctors/finish", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextDoctor != nil {
				c.Set(middleware.ContextKeyDoctorID, tt.setContextDoctor)
			}

			mockUC := &mockDoctorUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewDoctorHandler(mockUC)
			err := handler.FinishConsultation(c)
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

func TestDoctorHandler_GetWorkspace(t *testing.T) {
	tests := []struct {
		name             string
		setContextDoctor *string
		mockSetup        func(uc *mockDoctorUseCase)
		wantStatus       int
		wantBodySubstr   string
	}{
		{
			name:             "Missing doctor ID in context returns 403",
			setContextDoctor: nil,
			wantStatus:       http.StatusForbidden,
			wantBodySubstr:   "Doctor profile required",
		},
		{
			name:             "Doctor not found returns 404",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e999"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.getWorkspaceFunc = func(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
					return nil, domain.ErrDoctorNotFound
				}
			},
			wantStatus:     http.StatusNotFound,
			wantBodySubstr: "Doctor not found",
		},
		{
			name:             "UseCase returns ErrInvalidInput -> 400",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.getWorkspaceFunc = func(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			wantStatus:     http.StatusBadRequest,
			wantBodySubstr: "Invalid doctor ID",
		},
		{
			name:             "UseCase returns internal error -> 500",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.getWorkspaceFunc = func(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus:     http.StatusInternalServerError,
			wantBodySubstr: "Internal server error",
		},
		{
			name:             "Success returns 200 OK with workspace data",
			setContextDoctor: strPtr("01919df4-8e3b-7412-a1f9-90b567c9e101"),
			mockSetup: func(uc *mockDoctorUseCase) {
				uc.getWorkspaceFunc = func(ctx context.Context, doctorID string) (*domain.DoctorWorkspace, error) {
					return &domain.DoctorWorkspace{
						DoctorID:            "01919df4-8e3b-7412-a1f9-90b567c9e101",
						DoctorName:          "Dr. Sarah Adams",
						AvgConsultationTime: 3,
						IsOnline:            true,
						Status:              domain.DoctorStatusAvailable,
					}, nil
				}
			},
			wantStatus:     http.StatusOK,
			wantBodySubstr: `"doctor_name":"Dr. Sarah Adams"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/doctors/workspace", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setContextDoctor != nil {
				c.Set(middleware.ContextKeyDoctorID, tt.setContextDoctor)
			}

			mockUC := &mockDoctorUseCase{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockUC)
			}

			handler := NewDoctorHandler(mockUC)
			err := handler.GetWorkspace(c)
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

func TestDoctorHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	handler := NewDoctorHandler(&mockDoctorUseCase{})

	dummyMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	handler.RegisterRoutes(e, dummyMW, dummyMW)

	routes := e.Routes()
	expectedRoutes := map[string]string{
		"/api/doctors/status":    http.MethodPost,
		"/api/doctors/call-next": http.MethodPost,
		"/api/doctors/finish":    http.MethodPost,
		"/api/doctors/workspace": http.MethodGet,
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
