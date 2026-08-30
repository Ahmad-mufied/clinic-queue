package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepositoryPort struct {
	findByUsernameFunc func(ctx context.Context, username string) (*domain.User, error)
	findByIDFunc       func(ctx context.Context, id int) (*domain.User, error)
	createUserFunc     func(ctx context.Context, user *domain.User) (*domain.User, error)
}

func (m *mockUserRepositoryPort) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.findByUsernameFunc != nil {
		return m.findByUsernameFunc(ctx, username)
	}
	return nil, nil
}

func (m *mockUserRepositoryPort) FindByID(ctx context.Context, id int) (*domain.User, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepositoryPort) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, user)
	}
	return nil, nil
}

func TestAuthUseCase_Login(t *testing.T) {
	passHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	docID := 1

	tests := []struct {
		name        string
		req         inbound.LoginRequest
		mockSetup   func(m *mockUserRepositoryPort)
		wantErr     bool
		expectedErr error
		checkResp   func(t *testing.T, resp *inbound.AuthResponse)
	}{
		{
			name: "Doctor Login Success",
			req:  inbound.LoginRequest{Username: "doctor_a", Password: "password123"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return &domain.User{
						ID:           1,
						Username:     "doctor_a",
						PasswordHash: string(passHash),
						Name:         "Dr. Sarah Adams",
						Role:         domain.RoleDoctor,
						DoctorID:     &docID,
					}, nil
				}
			},
			wantErr: false,
			checkResp: func(t *testing.T, resp *inbound.AuthResponse) {
				if resp.Token == "" {
					t.Errorf("expected non-empty token")
				}
				if resp.User == nil || resp.User.Role != domain.RoleDoctor {
					t.Errorf("expected doctor user in response, got %+v", resp.User)
				}
			},
		},
		{
			name: "Patient Login Success",
			req:  inbound.LoginRequest{Username: "patient_john", Password: "password123"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return &domain.User{
						ID:           3,
						Username:     "patient_john",
						PasswordHash: string(passHash),
						Name:         "John Doe",
						Role:         domain.RolePatient,
					}, nil
				}
			},
			wantErr: false,
			checkResp: func(t *testing.T, resp *inbound.AuthResponse) {
				if resp.Token == "" {
					t.Errorf("expected non-empty token")
				}
				if resp.User == nil || resp.User.Role != domain.RolePatient {
					t.Errorf("expected patient user in response, got %+v", resp.User)
				}
			},
		},
		{
			name: "Admin Login Success",
			req:  inbound.LoginRequest{Username: "admin", Password: "password123"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return &domain.User{
						ID:           5,
						Username:     "admin",
						PasswordHash: string(passHash),
						Name:         "Clinic Administrator",
						Role:         domain.RoleAdmin,
					}, nil
				}
			},
			wantErr: false,
			checkResp: func(t *testing.T, resp *inbound.AuthResponse) {
				if resp.Token == "" {
					t.Errorf("expected non-empty token")
				}
				if resp.User == nil || resp.User.Role != domain.RoleAdmin {
					t.Errorf("expected admin user in response, got %+v", resp.User)
				}
			},
		},
		{
			name:        "Empty Username",
			req:         inbound.LoginRequest{Username: "", Password: "password123"},
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:        "Empty Password",
			req:         inbound.LoginRequest{Username: "doctor_a", Password: ""},
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name: "User Not Found",
			req:  inbound.LoginRequest{Username: "unknown_user", Password: "password123"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, nil
				}
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidCredentials,
		},
		{
			name: "Password Mismatch",
			req:  inbound.LoginRequest{Username: "doctor_a", Password: "wrongpassword"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return &domain.User{
						ID:           1,
						Username:     "doctor_a",
						PasswordHash: string(passHash),
						Name:         "Dr. Sarah Adams",
						Role:         domain.RoleDoctor,
					}, nil
				}
			},
			wantErr:     true,
			expectedErr: domain.ErrInvalidCredentials,
		},
		{
			name: "Database Error on FindByUsername",
			req:  inbound.LoginRequest{Username: "doctor_a", Password: "password123"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, errors.New("db connection failure")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepositoryPort{}
			tt.mockSetup(repo)

			uc := NewAuthUseCase(repo, "test-secret-key-12345", 24*time.Hour)
			resp, err := uc.Login(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResp != nil {
					tt.checkResp(t, resp)
				}
			}
		})
	}
}

func TestAuthUseCase_Register(t *testing.T) {
	tests := []struct {
		name        string
		req         inbound.RegisterRequest
		mockSetup   func(m *mockUserRepositoryPort)
		wantErr     bool
		expectedErr error
		checkResp   func(t *testing.T, resp *inbound.AuthResponse)
	}{
		{
			name: "Register Patient Success",
			req:  inbound.RegisterRequest{Username: "new_patient", Password: "password123", Name: "New Patient"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, nil
				}
				m.createUserFunc = func(ctx context.Context, user *domain.User) (*domain.User, error) {
					user.ID = 10
					return user, nil
				}
			},
			wantErr: false,
			checkResp: func(t *testing.T, resp *inbound.AuthResponse) {
				if resp.Token == "" {
					t.Errorf("expected non-empty token")
				}
				if resp.User == nil || resp.User.Role != domain.RolePatient || resp.User.ID != 10 {
					t.Errorf("expected patient user with ID 10, got %+v", resp.User)
				}
			},
		},
		{
			name:        "Empty Username",
			req:         inbound.RegisterRequest{Username: "", Password: "password123", Name: "New Patient"},
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:        "Empty Password",
			req:         inbound.RegisterRequest{Username: "new_patient", Password: "", Name: "New Patient"},
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:        "Empty Name",
			req:         inbound.RegisterRequest{Username: "new_patient", Password: "password123", Name: ""},
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name: "Username Already Taken",
			req:  inbound.RegisterRequest{Username: "existing_patient", Password: "password123", Name: "Existing Patient"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return &domain.User{ID: 2, Username: "existing_patient"}, nil
				}
			},
			wantErr:     true,
			expectedErr: domain.ErrUsernameTaken,
		},
		{
			name: "Password Too Long (> 72 bytes)",
			req:  inbound.RegisterRequest{Username: "new_patient", Password: "this_is_an_extremely_long_password_that_exceeds_seventy_two_bytes_limit_for_bcrypt_algorithm_0123456789", Name: "Patient"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, nil
				}
			},
			wantErr: true,
		},
		{
			name: "Database Error on FindByUsername",
			req:  inbound.RegisterRequest{Username: "patient_err", Password: "password123", Name: "Patient Err"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, errors.New("db query error")
				}
			},
			wantErr: true,
		},
		{
			name: "Database Error on CreateUser",
			req:  inbound.RegisterRequest{Username: "patient_create_err", Password: "password123", Name: "Patient Create Err"},
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByUsernameFunc = func(ctx context.Context, username string) (*domain.User, error) {
					return nil, nil
				}
				m.createUserFunc = func(ctx context.Context, user *domain.User) (*domain.User, error) {
					return nil, errors.New("insert failure")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepositoryPort{}
			tt.mockSetup(repo)

			uc := NewAuthUseCase(repo, "test-secret-key-12345", 24*time.Hour)
			resp, err := uc.Register(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResp != nil {
					tt.checkResp(t, resp)
				}
			}
		})
	}
}

func TestAuthUseCase_GetProfile(t *testing.T) {
	tests := []struct {
		name        string
		userID      int
		mockSetup   func(m *mockUserRepositoryPort)
		wantErr     bool
		expectedErr error
		checkUser   func(t *testing.T, user *domain.User)
	}{
		{
			name:   "Get Profile Success",
			userID: 1,
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByIDFunc = func(ctx context.Context, id int) (*domain.User, error) {
					return &domain.User{
						ID:       1,
						Username: "doctor_a",
						Name:     "Dr. Sarah Adams",
						Role:     domain.RoleDoctor,
					}, nil
				}
			},
			wantErr: false,
			checkUser: func(t *testing.T, user *domain.User) {
				if user == nil || user.ID != 1 || user.Username != "doctor_a" {
					t.Errorf("unexpected user object: %+v", user)
				}
			},
		},
		{
			name:        "Invalid User ID (<= 0)",
			userID:      0,
			mockSetup:   func(m *mockUserRepositoryPort) {},
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:   "User Not Found",
			userID: 999,
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByIDFunc = func(ctx context.Context, id int) (*domain.User, error) {
					return nil, nil
				}
			},
			wantErr:     true,
			expectedErr: domain.ErrUserNotFound,
		},
		{
			name:   "Database Error on FindByID",
			userID: 1,
			mockSetup: func(m *mockUserRepositoryPort) {
				m.findByIDFunc = func(ctx context.Context, id int) (*domain.User, error) {
					return nil, errors.New("db query error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepositoryPort{}
			tt.mockSetup(repo)

			uc := NewAuthUseCase(repo, "test-secret-key-12345", 24*time.Hour)
			user, err := uc.GetProfile(context.Background(), tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkUser != nil {
					tt.checkUser(t, user)
				}
			}
		})
	}
}

type mockAuthEventPub struct {
	published []string
}

func (m *mockAuthEventPub) PublishEvent(ctx context.Context, eventType string, payload any) error {
	m.published = append(m.published, eventType)
	return nil
}

func (m *mockAuthEventPub) Close() error {
	return nil
}

func TestAuthUseCase_WithEventPublisher(t *testing.T) {
	passHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo := &mockUserRepositoryPort{
		findByUsernameFunc: func(ctx context.Context, username string) (*domain.User, error) {
			if username == "new_user" {
				return nil, nil
			}
			return &domain.User{
				ID:           1,
				Username:     "existing_user",
				PasswordHash: string(passHash),
				Name:         "Existing User",
				Role:         domain.RoleDoctor,
			}, nil
		},
		createUserFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			user.ID = 2
			return user, nil
		},
	}

	ep := &mockAuthEventPub{}
	uc := NewAuthUseCase(repo, "secret", time.Hour, ep)

	// Test Login with EventPub
	loginResp, err := uc.Login(context.Background(), inbound.LoginRequest{
		Username: "existing_user",
		Password: "password123",
	})
	if err != nil || loginResp == nil {
		t.Fatalf("login failed: %v", err)
	}

	// Test Register with EventPub
	regResp, err := uc.Register(context.Background(), inbound.RegisterRequest{
		Username: "new_user",
		Password: "password123",
		Name:     "New User",
	})
	if err != nil || regResp == nil {
		t.Fatalf("register failed: %v", err)
	}

	if len(ep.published) != 2 || ep.published[0] != "AUTH_LOGIN" || ep.published[1] != "AUTH_REGISTER" {
		t.Errorf("unexpected published events: %v", ep.published)
	}
}
