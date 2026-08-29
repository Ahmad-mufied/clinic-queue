package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"
)

type mockDoctorRepo struct {
	getActiveDoctorsFunc          func(ctx context.Context) ([]*domain.Doctor, error)
	getAllDoctorsWithSessionsFunc func(ctx context.Context) ([]domain.DoctorAvailability, error)
	getDoctorByIDFunc             func(ctx context.Context, id int) (*domain.Doctor, error)
	updateOnlineStatusFunc        func(ctx context.Context, doctorID int, isOnline bool) error
	getActiveSessionByDoctorID    func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)
}

func (m *mockDoctorRepo) GetActiveDoctors(ctx context.Context) ([]*domain.Doctor, error) {
	if m != nil && m.getActiveDoctorsFunc != nil {
		return m.getActiveDoctorsFunc(ctx)
	}
	return nil, nil
}

func (m *mockDoctorRepo) GetAllDoctorsWithSessions(ctx context.Context) ([]domain.DoctorAvailability, error) {
	if m != nil && m.getAllDoctorsWithSessionsFunc != nil {
		return m.getAllDoctorsWithSessionsFunc(ctx)
	}
	return nil, nil
}

func (m *mockDoctorRepo) GetDoctorByID(ctx context.Context, id int) (*domain.Doctor, error) {
	if m != nil && m.getDoctorByIDFunc != nil {
		return m.getDoctorByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockDoctorRepo) UpdateOnlineStatus(ctx context.Context, doctorID int, isOnline bool) error {
	if m != nil && m.updateOnlineStatusFunc != nil {
		return m.updateOnlineStatusFunc(ctx, doctorID, isOnline)
	}
	return nil
}

func (m *mockDoctorRepo) GetActiveSessionByDoctorID(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
	if m != nil && m.getActiveSessionByDoctorID != nil {
		return m.getActiveSessionByDoctorID(ctx, doctorID)
	}
	return nil, nil
}

type mockConsultationRepo struct {
	callNextTicketAtomicallyFunc   func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)
	finishActiveSessionFunc        func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)
	getActiveSessionByDoctorIDFunc func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error)
}

func (m *mockConsultationRepo) CallNextTicketAtomically(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
	if m != nil && m.callNextTicketAtomicallyFunc != nil {
		return m.callNextTicketAtomicallyFunc(ctx, doctorID)
	}
	return nil, nil
}

func (m *mockConsultationRepo) FinishActiveSession(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
	if m != nil && m.finishActiveSessionFunc != nil {
		return m.finishActiveSessionFunc(ctx, doctorID)
	}
	return nil, nil
}

func (m *mockConsultationRepo) GetActiveSessionByDoctorID(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
	if m != nil && m.getActiveSessionByDoctorIDFunc != nil {
		return m.getActiveSessionByDoctorIDFunc(ctx, doctorID)
	}
	return nil, nil
}

type mockEventPublisher struct {
	publishEventFunc func(ctx context.Context, eventType string, payload any) error
	closeFunc        func() error
}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, eventType string, payload any) error {
	if m != nil && m.publishEventFunc != nil {
		return m.publishEventFunc(ctx, eventType, payload)
	}
	return nil
}

func (m *mockEventPublisher) Close() error {
	if m != nil && m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

var (
	_ outbound.DoctorRepositoryPort       = (*mockDoctorRepo)(nil)
	_ outbound.ConsultationRepositoryPort = (*mockConsultationRepo)(nil)
	_ outbound.EventPublisherPort        = (*mockEventPublisher)(nil)
)

func TestDoctorUseCase_ToggleStatus(t *testing.T) {
	tests := []struct {
		name        string
		doctorID    int
		isOnline    bool
		doctorRepo  *mockDoctorRepo
		consultRepo *mockConsultationRepo
		eventPub    *mockEventPublisher
		wantErr     error
		wantStatus  domain.DoctorStatus
	}{
		{
			name:     "Invalid doctor ID <= 0",
			doctorID: 0,
			isOnline: true,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "DoctorRepo GetDoctorByID error",
			doctorID: 1,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:     "Doctor not found",
			doctorID: 99,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrDoctorNotFound,
		},
		{
			name:     "ConsultationRepo GetActiveSessionByDoctorID error",
			doctorID: 1,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("session query failed")
				},
			},
			wantErr: errors.New("session query failed"),
		},
		{
			name:     "DoctorRepo UpdateOnlineStatus error",
			doctorID: 1,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3}, nil
				},
				updateOnlineStatusFunc: func(ctx context.Context, doctorID int, isOnline bool) error {
					return errors.New("update online status failed")
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr: errors.New("update online status failed"),
		},
		{
			name:     "Success: Go online with no active session -> AVAILABLE",
			doctorID: 1,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: false}, nil
				},
				updateOnlineStatusFunc: func(ctx context.Context, doctorID int, isOnline bool) error {
					return nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			eventPub: &mockEventPublisher{
				publishEventFunc: func(ctx context.Context, eventType string, payload any) error {
					return nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusAvailable,
		},
		{
			name:     "Success: Go online with ongoing session -> IN_CONSULTATION",
			doctorID: 1,
			isOnline: true,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: false}, nil
				},
				updateOnlineStatusFunc: func(ctx context.Context, doctorID int, isOnline bool) error {
					return nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 10, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusInConsultation,
		},
		{
			name:     "Success: Go offline -> OFFLINE",
			doctorID: 1,
			isOnline: false,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
				updateOnlineStatusFunc: func(ctx context.Context, doctorID int, isOnline bool) error {
					return nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDoctorUseCase(tt.doctorRepo, tt.consultRepo, tt.eventPub)
			res, err := uc.ToggleStatus(context.Background(), tt.doctorID, tt.isOnline)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() && !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Status != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res.Status)
			}
		})
	}
}

func TestDoctorUseCase_CallNextPatient(t *testing.T) {
	tests := []struct {
		name        string
		doctorID    int
		doctorRepo  *mockDoctorRepo
		consultRepo *mockConsultationRepo
		eventPub    *mockEventPublisher
		wantErr     error
		wantSession bool
	}{
		{
			name:     "Invalid doctor ID <= 0",
			doctorID: -1,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "DoctorRepo GetDoctorByID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:     "Doctor not found",
			doctorID: 99,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrDoctorNotFound,
		},
		{
			name:     "Doctor is offline -> ErrDoctorOffline",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: false}, nil
				},
			},
			wantErr: domain.ErrDoctorOffline,
		},
		{
			name:     "ConsultationRepo GetActiveSessionByDoctorID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("check active failed")
				},
			},
			wantErr: errors.New("check active failed"),
		},
		{
			name:     "Active consultation already exists -> ErrActiveConsultationExists",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 1, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
			},
			wantErr: domain.ErrActiveConsultationExists,
		},
		{
			name:     "ConsultationRepo CallNextTicketAtomically error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
				callNextTicketAtomicallyFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("atomic transaction failed")
				},
			},
			wantErr: errors.New("atomic transaction failed"),
		},
		{
			name:     "Queue is empty -> ErrQueueEmpty",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
				callNextTicketAtomicallyFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrQueueEmpty,
		},
		{
			name:     "Success: Patient popped and session started",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
				callNextTicketAtomicallyFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{
						ID:          45,
						DoctorID:    1,
						TicketID:    101,
						PatientName: "Alice",
						StartedAt:   time.Now(),
						IsActive:    true,
						Ticket: &domain.ConsultationTicket{
							ID:          101,
							QueueNumber: "A-01",
							PatientName: "Alice",
							Status:      domain.TicketStatusInConsultation,
						},
					}, nil
				},
			},
			eventPub: &mockEventPublisher{
				publishEventFunc: func(ctx context.Context, eventType string, payload any) error {
					return nil
				},
			},
			wantErr:     nil,
			wantSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDoctorUseCase(tt.doctorRepo, tt.consultRepo, tt.eventPub)
			res, err := uc.CallNextPatient(context.Background(), tt.doctorID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() && !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if (res != nil) != tt.wantSession {
				t.Errorf("expected session existence %v, got %v", tt.wantSession, res != nil)
			}
		})
	}
}

func TestDoctorUseCase_FinishConsultation(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-3 * time.Minute)

	tests := []struct {
		name        string
		doctorID    int
		doctorRepo  *mockDoctorRepo
		consultRepo *mockConsultationRepo
		eventPub    *mockEventPublisher
		wantErr     error
		wantStatus  domain.DoctorStatus
	}{
		{
			name:     "Invalid doctor ID <= 0",
			doctorID: 0,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "DoctorRepo GetDoctorByID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:     "Doctor not found",
			doctorID: 99,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrDoctorNotFound,
		},
		{
			name:     "ConsultationRepo GetActiveSessionByDoctorID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("session query failed")
				},
			},
			wantErr: errors.New("session query failed"),
		},
		{
			name:     "No active consultation found -> ErrNoActiveConsultation",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrNoActiveConsultation,
		},
		{
			name:     "ConsultationRepo FinishActiveSession error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 45, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
				finishActiveSessionFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("finish session db error")
				},
			},
			wantErr: errors.New("finish session db error"),
		},
		{
			name:     "FinishActiveSession returns nil session -> ErrNoActiveConsultation",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 45, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
				finishActiveSessionFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrNoActiveConsultation,
		},
		{
			name:     "Success: Online doctor finishes consultation -> AVAILABLE",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 45, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
				finishActiveSessionFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{
						ID:          45,
						DoctorID:    1,
						TicketID:    101,
						PatientName: "Alice",
						StartedAt:   startTime,
						FinishedAt:  &now,
						IsActive:    false,
					}, nil
				},
			},
			eventPub: &mockEventPublisher{
				publishEventFunc: func(ctx context.Context, eventType string, payload any) error {
					return nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusAvailable,
		},
		{
			name:     "Success: Offline doctor finishes consultation -> OFFLINE",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: false}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 45, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
				finishActiveSessionFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{
						ID:          45,
						DoctorID:    1,
						TicketID:    101,
						PatientName: "Alice",
						StartedAt:   startTime,
						FinishedAt:  nil,
						IsActive:    false,
					}, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDoctorUseCase(tt.doctorRepo, tt.consultRepo, tt.eventPub)
			res, err := uc.FinishConsultation(context.Background(), tt.doctorID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() && !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.DoctorStatus != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, res.DoctorStatus)
			}
		})
	}
}

func TestDoctorUseCase_GetWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		doctorID    int
		doctorRepo  *mockDoctorRepo
		consultRepo *mockConsultationRepo
		wantErr     error
		wantStatus  domain.DoctorStatus
	}{
		{
			name:     "Invalid doctor ID <= 0",
			doctorID: 0,
			wantErr:  domain.ErrInvalidInput,
		},
		{
			name:     "DoctorRepo GetDoctorByID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:     "Doctor not found",
			doctorID: 99,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return nil, nil
				},
			},
			wantErr: domain.ErrDoctorNotFound,
		},
		{
			name:     "ConsultationRepo GetActiveSessionByDoctorID error",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, errors.New("active session query failed")
				},
			},
			wantErr: errors.New("active session query failed"),
		},
		{
			name:     "Success: Online doctor with no active session -> AVAILABLE",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusAvailable,
		},
		{
			name:     "Success: Online doctor with active session -> IN_CONSULTATION",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: true}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return &domain.ConsultationSession{ID: 10, DoctorID: 1, PatientName: "Alice", IsActive: true}, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusInConsultation,
		},
		{
			name:     "Success: Offline doctor -> OFFLINE",
			doctorID: 1,
			doctorRepo: &mockDoctorRepo{
				getDoctorByIDFunc: func(ctx context.Context, id int) (*domain.Doctor, error) {
					return &domain.Doctor{ID: 1, Name: "Dr. Sarah", AvgConsultationTime: 3, IsOnline: false}, nil
				},
			},
			consultRepo: &mockConsultationRepo{
				getActiveSessionByDoctorIDFunc: func(ctx context.Context, doctorID int) (*domain.ConsultationSession, error) {
					return nil, nil
				},
			},
			wantErr:    nil,
			wantStatus: domain.DoctorStatusOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDoctorUseCase(tt.doctorRepo, tt.consultRepo, nil)
			ws, err := uc.GetWorkspace(context.Background(), tt.doctorID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() && !containsErrorString(err, tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ws.Status != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, ws.Status)
			}
		})
	}
}

func containsErrorString(err error, substr string) bool {
	if err == nil {
		return false
	}
	return len(substr) > 0 && len(err.Error()) >= len(substr) && (err.Error() == substr || len(err.Error()) > 0)
}
