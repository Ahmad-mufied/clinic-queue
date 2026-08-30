package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

type mockAuthUseCase struct {
	loginFunc      func(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error)
	registerFunc   func(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error)
	getProfileFunc func(ctx context.Context, userID string) (*domain.User, error)
}

func (m *mockAuthUseCase) Login(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAuthUseCase) Register(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAuthUseCase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, userID)
	}
	return nil, nil
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	mockUC := &mockAuthUseCase{}
	handler := NewAuthHandler(mockUC)

	dummyAuthMW := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	dummyRbacMW := func(next echo.HandlerFunc) echo.HandlerFunc { return next }

	handler.RegisterRoutes(e, dummyAuthMW, dummyRbacMW)

	routes := e.Routes()
	expectedRoutes := map[string]string{
		"POST /api/auth/login":    "clinic-queue/internal/adapters/inbound/http.(*AuthHandler).Login-fm",
		"POST /api/auth/register": "clinic-queue/internal/adapters/inbound/http.(*AuthHandler).Register-fm",
		"GET /api/auth/me":        "clinic-queue/internal/adapters/inbound/http.(*AuthHandler).GetMe-fm",
	}

	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := expectedRoutes[key]; ok {
			delete(expectedRoutes, key)
		}
	}

	if len(expectedRoutes) > 0 {
		t.Errorf("missing expected routes: %+v", expectedRoutes)
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		payload      interface{}
		rawBody      string
		mockSetup    func(m *mockAuthUseCase)
		expectedCode int
	}{
		{
			name:    "Login 200 OK",
			payload: inbound.LoginRequest{Username: "doctor_a", Password: "password123"},
			mockSetup: func(m *mockAuthUseCase) {
				m.loginFunc = func(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
					return &inbound.AuthResponse{
						Token: "valid-jwt-token",
						User:  &domain.User{ID: "01919df4-8e3b-7412-a1f9-90b567c9e201", Username: "doctor_a", Role: domain.RoleDoctor},
					}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Login Bad JSON",
			rawBody:      `{"username": "doc", "password": `,
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Login Empty Username",
			payload:      inbound.LoginRequest{Username: "", Password: "password123"},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Login Empty Password",
			payload:      inbound.LoginRequest{Username: "doctor_a", Password: ""},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "Login Invalid Credentials",
			payload: inbound.LoginRequest{Username: "doctor_a", Password: "wrongpassword"},
			mockSetup: func(m *mockAuthUseCase) {
				m.loginFunc = func(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
					return nil, domain.ErrInvalidCredentials
				}
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:    "Login Invalid Input from UseCase",
			payload: inbound.LoginRequest{Username: "doc", Password: "pw"},
			mockSetup: func(m *mockAuthUseCase) {
				m.loginFunc = func(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "Login Internal Service Error",
			payload: inbound.LoginRequest{Username: "doctor_a", Password: "password123"},
			mockSetup: func(m *mockAuthUseCase) {
				m.loginFunc = func(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
					return nil, errors.New("unexpected database error")
				}
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var body []byte
			if tt.rawBody != "" {
				body = []byte(tt.rawBody)
			} else {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockAuthUseCase{}
			tt.mockSetup(mockUC)

			handler := NewAuthHandler(mockUC)
			_ = handler.Login(c)

			if rec.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name         string
		payload      interface{}
		rawBody      string
		mockSetup    func(m *mockAuthUseCase)
		expectedCode int
	}{
		{
			name:    "Register 201 Created",
			payload: inbound.RegisterRequest{Username: "new_user", Password: "password123", Name: "New User"},
			mockSetup: func(m *mockAuthUseCase) {
				m.registerFunc = func(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
					return &inbound.AuthResponse{
						Token: "valid-jwt-token",
						User:  &domain.User{ID: "01919df4-8e3b-7412-a1f9-90b567c9e205", Username: "new_user", Name: "New User", Role: domain.RolePatient},
					}, nil
				}
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:         "Register Bad JSON",
			rawBody:      `{"username": "invalid`,
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Register Empty Username",
			payload:      inbound.RegisterRequest{Username: "", Password: "password123", Name: "Name"},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Register Empty Password",
			payload:      inbound.RegisterRequest{Username: "user", Password: "", Name: "Name"},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Register Empty Name",
			payload:      inbound.RegisterRequest{Username: "user", Password: "password123", Name: ""},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "Register Username Conflict",
			payload: inbound.RegisterRequest{Username: "existing_user", Password: "password123", Name: "Name"},
			mockSetup: func(m *mockAuthUseCase) {
				m.registerFunc = func(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
					return nil, domain.ErrUsernameTaken
				}
			},
			expectedCode: http.StatusConflict,
		},
		{
			name:    "Register Invalid Input from UseCase",
			payload: inbound.RegisterRequest{Username: "u", Password: "p", Name: "n"},
			mockSetup: func(m *mockAuthUseCase) {
				m.registerFunc = func(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:    "Register Internal Server Error",
			payload: inbound.RegisterRequest{Username: "new_user", Password: "password123", Name: "Name"},
			mockSetup: func(m *mockAuthUseCase) {
				m.registerFunc = func(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
					return nil, errors.New("db insert failure")
				}
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var body []byte
			if tt.rawBody != "" {
				body = []byte(tt.rawBody)
			} else {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &mockAuthUseCase{}
			tt.mockSetup(mockUC)

			handler := NewAuthHandler(mockUC)
			_ = handler.Register(c)

			if rec.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthHandler_GetMe(t *testing.T) {
	tests := []struct {
		name         string
		contextSetup func(c echo.Context)
		mockSetup    func(m *mockAuthUseCase)
		expectedCode int
	}{
		{
			name: "GetMe 200 OK",
			contextSetup: func(c echo.Context) {
				c.Set(middleware.ContextKeyUserID, "01919df4-8e3b-7412-a1f9-90b567c9e201")
			},
			mockSetup: func(m *mockAuthUseCase) {
				m.getProfileFunc = func(ctx context.Context, userID string) (*domain.User, error) {
					return &domain.User{ID: "01919df4-8e3b-7412-a1f9-90b567c9e201", Username: "doctor_a", Name: "Dr. Sarah Adams", Role: domain.RoleDoctor}, nil
				}
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "GetMe Unauthenticated (No context user_id)",
			contextSetup: func(c echo.Context) {},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "GetMe Invalid User ID (empty)",
			contextSetup: func(c echo.Context) {
				c.Set(middleware.ContextKeyUserID, "")
			},
			mockSetup:    func(m *mockAuthUseCase) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "GetMe User Not Found",
			contextSetup: func(c echo.Context) {
				c.Set(middleware.ContextKeyUserID, "01919df4-8e3b-7412-a1f9-90b567c9e999")
			},
			mockSetup: func(m *mockAuthUseCase) {
				m.getProfileFunc = func(ctx context.Context, userID string) (*domain.User, error) {
					return nil, domain.ErrUserNotFound
				}
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name: "GetMe Invalid Input from UseCase",
			contextSetup: func(c echo.Context) {
				c.Set(middleware.ContextKeyUserID, "01919df4-8e3b-7412-a1f9-90b567c9e201")
			},
			mockSetup: func(m *mockAuthUseCase) {
				m.getProfileFunc = func(ctx context.Context, userID string) (*domain.User, error) {
					return nil, domain.ErrInvalidInput
				}
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "GetMe Internal Server Error",
			contextSetup: func(c echo.Context) {
				c.Set(middleware.ContextKeyUserID, "01919df4-8e3b-7412-a1f9-90b567c9e201")
			},
			mockSetup: func(m *mockAuthUseCase) {
				m.getProfileFunc = func(ctx context.Context, userID string) (*domain.User, error) {
					return nil, errors.New("db error")
				}
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.contextSetup(c)

			mockUC := &mockAuthUseCase{}
			tt.mockSetup(mockUC)

			handler := NewAuthHandler(mockUC)
			_ = handler.GetMe(c)

			if rec.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}
		})
	}
}
