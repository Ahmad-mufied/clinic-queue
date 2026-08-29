package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"
)

// --- Mock Implementations ---

type mockQueueRepoPort struct {
	createTicketFunc                  func(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error)
	findActiveTicketByUserIDFunc      func(ctx context.Context, userID int) (*domain.QueueTicket, error)
	findActiveTicketByPatientNameFunc func(ctx context.Context, patientName string) (*domain.QueueTicket, error)
	findByIDFunc                      func(ctx context.Context, id int) (*domain.QueueTicket, error)
	getWaitingTicketsFunc             func(ctx context.Context) ([]*domain.QueueTicket, error)
	countWaitingAheadFunc             func(ctx context.Context, createdAt time.Time) (int, error)
	getNextQueueNumberFunc            func(ctx context.Context) (string, error)
}

func (m *mockQueueRepoPort) CreateTicket(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
	if m.createTicketFunc != nil {
		return m.createTicketFunc(ctx, ticket)
	}
	return ticket, nil
}

func (m *mockQueueRepoPort) FindActiveTicketByUserID(ctx context.Context, userID int) (*domain.QueueTicket, error) {
	if m.findActiveTicketByUserIDFunc != nil {
		return m.findActiveTicketByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockQueueRepoPort) FindActiveTicketByPatientName(ctx context.Context, patientName string) (*domain.QueueTicket, error) {
	if m.findActiveTicketByPatientNameFunc != nil {
		return m.findActiveTicketByPatientNameFunc(ctx, patientName)
	}
	return nil, nil
}

func (m *mockQueueRepoPort) FindByID(ctx context.Context, id int) (*domain.QueueTicket, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockQueueRepoPort) GetWaitingTickets(ctx context.Context) ([]*domain.QueueTicket, error) {
	if m.getWaitingTicketsFunc != nil {
		return m.getWaitingTicketsFunc(ctx)
	}
	return nil, nil
}

func (m *mockQueueRepoPort) CountWaitingAhead(ctx context.Context, createdAt time.Time) (int, error) {
	if m.countWaitingAheadFunc != nil {
		return m.countWaitingAheadFunc(ctx, createdAt)
	}
	return 0, nil
}

func (m *mockQueueRepoPort) GetNextQueueNumber(ctx context.Context) (string, error) {
	if m.getNextQueueNumberFunc != nil {
		return m.getNextQueueNumberFunc(ctx)
	}
	return "A-01", nil
}

type mockDoctorRepoPort struct {
	getActiveDoctorsFunc          func(ctx context.Context) ([]*domain.Doctor, error)
	getAllDoctorsWithSessionsFunc func(ctx context.Context) ([]domain.DoctorAvailability, error)
	getDoctorByIDFunc             func(ctx context.Context, id int) (*domain.Doctor, error)
}

func (m *mockDoctorRepoPort) GetActiveDoctors(ctx context.Context) ([]*domain.Doctor, error) {
	if m.getActiveDoctorsFunc != nil {
		return m.getActiveDoctorsFunc(ctx)
	}
	return nil, nil
}

func (m *mockDoctorRepoPort) GetAllDoctorsWithSessions(ctx context.Context) ([]domain.DoctorAvailability, error) {
	if m.getAllDoctorsWithSessionsFunc != nil {
		return m.getAllDoctorsWithSessionsFunc(ctx)
	}
	return nil, nil
}

func (m *mockDoctorRepoPort) GetDoctorByID(ctx context.Context, id int) (*domain.Doctor, error) {
	if m.getDoctorByIDFunc != nil {
		return m.getDoctorByIDFunc(ctx, id)
	}
	return nil, nil
}

type mockEventPubPort struct {
	publishEventFunc func(ctx context.Context, eventType string, payload any) error
	closeFunc        func() error
}

func (m *mockEventPubPort) PublishEvent(ctx context.Context, eventType string, payload any) error {
	if m.publishEventFunc != nil {
		return m.publishEventFunc(ctx, eventType, payload)
	}
	return nil
}

func (m *mockEventPubPort) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// --- Unit Tests ---

func intPtr(v int) *int {
	return &v
}

func TestQueueUseCase_JoinQueue(t *testing.T) {
	errDB := errors.New("database error")
	userIDVal := 10

	tests := []struct {
		name         string
		userID       *int
		patientName  string
		setupMocks   func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort)
		wantErrIs    error
		wantErrStr   string
		wantQueueNum string
		wantWaitMin  *int
		wantNotice   string
		wantPos      int
		wantAhead    int
	}{
		{
			name:        "Empty patient name returns ErrInvalidInput",
			userID:      &userIDVal,
			patientName: "   ",
			setupMocks:  func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {},
			wantErrIs:   domain.ErrInvalidInput,
		},
		{
			name:        "Active ticket exists for user returns ErrActiveTicketExists",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 1, QueueNumber: "A-01", Status: domain.TicketStatusWaiting}, nil
				}
			},
			wantErrIs: domain.ErrActiveTicketExists,
		},
		{
			name:        "Database error on FindActiveTicketByUserID",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "check active user ticket",
		},
		{
			name:        "Active ticket exists with same patient name returns ErrActiveTicketExists",
			userID:      nil,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				q.findActiveTicketByPatientNameFunc = func(ctx context.Context, name string) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 2, QueueNumber: "A-02", Status: domain.TicketStatusWaiting}, nil
				}
			},
			wantErrIs: domain.ErrActiveTicketExists,
		},
		{
			name:        "Database error on FindActiveTicketByPatientName",
			userID:      nil,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				q.findActiveTicketByPatientNameFunc = func(ctx context.Context, name string) (*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "check active name ticket",
		},
		{
			name:        "Database error on GetAllDoctorsWithSessions",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get all doctors",
		},
		{
			name:        "No doctors registered in clinic returns ErrNoDoctorsAvailable",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{}, nil
				}
			},
			wantErrIs: domain.ErrNoDoctorsAvailable,
		},
		{
			name:        "Database error on GetActiveDoctors",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc A", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get active doctors",
		},
		{
			name:        "Database error on GetWaitingTickets",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc A", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{{ID: 1, Name: "Doc A", AvgConsultationTime: 3, IsOnline: true}}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get waiting tickets",
		},
		{
			name:        "Error in CalculateEstimatedWaitingTime (invalid doctor data)",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc Invalid", AvgConsultationTimeMinutes: 0, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{{ID: 1, Name: "Doc Invalid", AvgConsultationTime: 0, IsOnline: true}}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{{ID: 1, QueueNumber: "A-01"}}, nil
				}
			},
			wantErrStr: "calculate estimated waiting time",
		},
		{
			name:        "Database error on GetNextQueueNumber",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc A", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{{ID: 1, Name: "Doc A", AvgConsultationTime: 3, IsOnline: true}}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{}, nil
				}
				q.getNextQueueNumberFunc = func(ctx context.Context) (string, error) {
					return "", errDB
				}
			},
			wantErrStr: "generate next queue number",
		},
		{
			name:        "Database error on CreateTicket",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc A", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{{ID: 1, Name: "Doc A", AvgConsultationTime: 3, IsOnline: true}}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{}, nil
				}
				q.getNextQueueNumberFunc = func(ctx context.Context) (string, error) {
					return "A-01", nil
				}
				q.createTicketFunc = func(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "create queue ticket",
		},
		{
			name:        "Success: John joins when doctors exist but are all offline (wait time is null with notice)",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doc A", AvgConsultationTimeMinutes: 3, IsOnline: false}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{}, nil
				}
				q.getNextQueueNumberFunc = func(ctx context.Context) (string, error) {
					return "A-01", nil
				}
				q.createTicketFunc = func(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
					t := *ticket
					t.ID = 101
					t.CreatedAt = time.Now()
					return &t, nil
				}
			},
			wantQueueNum: "A-01",
			wantWaitMin:  nil,
			wantNotice:   "Estimated wait time is currently unavailable because all doctors are offline / on break. Calculation will activate once a doctor starts duty.",
			wantPos:      1,
			wantAhead:    0,
		},
		{
			name:        "Success: John joins as 6th with 1 Doc (3m) idle (CS1 Q1)",
			userID:      &userIDVal,
			patientName: "John Doe",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doctor 1", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{
						{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true},
					}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{
						{ID: 1, QueueNumber: "A-01"},
						{ID: 2, QueueNumber: "A-02"},
						{ID: 3, QueueNumber: "A-03"},
						{ID: 4, QueueNumber: "A-04"},
						{ID: 5, QueueNumber: "A-05"},
					}, nil
				}
				q.getNextQueueNumberFunc = func(ctx context.Context) (string, error) {
					return "A-06", nil
				}
				q.createTicketFunc = func(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
					t := *ticket
					t.ID = 106
					t.CreatedAt = time.Now()
					return &t, nil
				}
				e.publishEventFunc = func(ctx context.Context, eventType string, payload any) error {
					return nil
				}
			},
			wantQueueNum: "A-06",
			wantWaitMin:  intPtr(15),
			wantNotice:   "",
			wantPos:      6,
			wantAhead:    5,
		},
		{
			name:        "Success: John joins without userID (Walk-in guest) & EventPub error handled gracefully",
			userID:      nil,
			patientName: "Anonymous Walkin",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort, e *mockEventPubPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{{ID: 1, Name: "Doctor 1", AvgConsultationTimeMinutes: 3, IsOnline: true}}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{
						{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true},
					}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{}, nil
				}
				q.getNextQueueNumberFunc = func(ctx context.Context) (string, error) {
					return "A-01", nil
				}
				q.createTicketFunc = func(ctx context.Context, ticket *domain.QueueTicket) (*domain.QueueTicket, error) {
					t := *ticket
					t.ID = 101
					t.CreatedAt = time.Now()
					return &t, nil
				}
				e.publishEventFunc = func(ctx context.Context, eventType string, payload any) error {
					return errors.New("nats timeout")
				}
			},
			wantQueueNum: "A-01",
			wantWaitMin:  intPtr(0),
			wantNotice:   "",
			wantPos:      1,
			wantAhead:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &mockQueueRepoPort{}
			mockDoc := &mockDoctorRepoPort{}
			mockEvent := &mockEventPubPort{}
			tt.setupMocks(mockQueue, mockDoc, mockEvent)

			uc := NewQueueUseCase(mockQueue, mockDoc, mockEvent)
			ticket, err := uc.JoinQueue(context.Background(), tt.userID, tt.patientName)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("JoinQueue() error = %v, wantErrIs %v", err, tt.wantErrIs)
				}
				return
			}

			if tt.wantErrStr != "" {
				if err == nil {
					t.Fatalf("JoinQueue() expected error containing %q, got nil", tt.wantErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("JoinQueue() unexpected error: %v", err)
			}

			if ticket.QueueNumber != tt.wantQueueNum {
				t.Errorf("JoinQueue() QueueNumber = %s, want %s", ticket.QueueNumber, tt.wantQueueNum)
			}
			if tt.wantWaitMin == nil {
				if ticket.EstimatedWaitTimeMinutes != nil {
					t.Errorf("JoinQueue() EstimatedWaitTimeMinutes = %v, want nil", *ticket.EstimatedWaitTimeMinutes)
				}
			} else {
				if ticket.EstimatedWaitTimeMinutes == nil || *ticket.EstimatedWaitTimeMinutes != *tt.wantWaitMin {
					t.Errorf("JoinQueue() EstimatedWaitTimeMinutes = %v, want %v", ticket.EstimatedWaitTimeMinutes, *tt.wantWaitMin)
				}
			}
			if ticket.Notice != tt.wantNotice {
				t.Errorf("JoinQueue() Notice = %q, want %q", ticket.Notice, tt.wantNotice)
			}
			if ticket.PositionInQueue != tt.wantPos {
				t.Errorf("JoinQueue() PositionInQueue = %d, want %d", ticket.PositionInQueue, tt.wantPos)
			}
			if ticket.AheadCount != tt.wantAhead {
				t.Errorf("JoinQueue() AheadCount = %d, want %d", ticket.AheadCount, tt.wantAhead)
			}
		})
	}
}

func TestQueueUseCase_GetMyTicket(t *testing.T) {
	errDB := errors.New("database error")
	now := time.Now()

	tests := []struct {
		name         string
		userID       int
		setupMocks   func(q *mockQueueRepoPort, d *mockDoctorRepoPort)
		wantErrIs    error
		wantErrStr   string
		wantQueueNum string
		wantWaitMin  *int
		wantNotice   string
		wantPos      int
		wantAhead    int
	}{
		{
			name:       "Invalid userID <= 0 returns ErrInvalidInput",
			userID:     0,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {},
			wantErrIs:  domain.ErrInvalidInput,
		},
		{
			name:   "Database error on FindActiveTicketByUserID",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "find active ticket",
		},
		{
			name:   "Ticket not found returns ErrTicketNotFound",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return nil, nil
				}
			},
			wantErrIs: domain.ErrTicketNotFound,
		},
		{
			name:   "Database error on CountWaitingAhead",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 5, QueueNumber: "A-05", Status: domain.TicketStatusWaiting, CreatedAt: now}, nil
				}
				q.countWaitingAheadFunc = func(ctx context.Context, createdAt time.Time) (int, error) {
					return 0, errDB
				}
			},
			wantErrStr: "count waiting ahead",
		},
		{
			name:   "Database error on GetActiveDoctors",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 5, QueueNumber: "A-05", Status: domain.TicketStatusWaiting, CreatedAt: now}, nil
				}
				q.countWaitingAheadFunc = func(ctx context.Context, createdAt time.Time) (int, error) {
					return 4, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get active doctors",
		},
		{
			name:   "Success with active doctors and recalculated wait time (4 ahead -> pos 5 -> 12m)",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 5, QueueNumber: "A-05", Status: domain.TicketStatusWaiting, CreatedAt: now}, nil
				}
				q.countWaitingAheadFunc = func(ctx context.Context, createdAt time.Time) (int, error) {
					return 4, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{
						{ID: 1, Name: "Doctor 1", AvgConsultationTime: 3, IsOnline: true},
					}, nil
				}
			},
			wantQueueNum: "A-05",
			wantWaitMin:  intPtr(12),
			wantNotice:   "",
			wantPos:      5,
			wantAhead:    4,
		},
		{
			name:   "Success when all doctors are offline (wait time is null with notice)",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 5, QueueNumber: "A-05", Status: domain.TicketStatusWaiting, CreatedAt: now}, nil
				}
				q.countWaitingAheadFunc = func(ctx context.Context, createdAt time.Time) (int, error) {
					return 2, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{}, nil
				}
			},
			wantQueueNum: "A-05",
			wantWaitMin:  nil,
			wantNotice:   "Estimated wait time is currently unavailable because all doctors are offline / on break. Calculation will activate once a doctor starts duty.",
			wantPos:      3,
			wantAhead:    2,
		},
		{
			name:   "Success for ticket in IN_CONSULTATION status (no waiting calculation)",
			userID: 10,
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				q.findActiveTicketByUserIDFunc = func(ctx context.Context, userID int) (*domain.QueueTicket, error) {
					return &domain.QueueTicket{ID: 3, QueueNumber: "A-03", Status: domain.TicketStatusInConsultation, CreatedAt: now}, nil
				}
			},
			wantQueueNum: "A-03",
			wantWaitMin:  nil,
			wantNotice:   "",
			wantPos:      0,
			wantAhead:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &mockQueueRepoPort{}
			mockDoc := &mockDoctorRepoPort{}
			tt.setupMocks(mockQueue, mockDoc)

			uc := NewQueueUseCase(mockQueue, mockDoc, nil)
			ticket, err := uc.GetMyTicket(context.Background(), tt.userID)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("GetMyTicket() error = %v, wantErrIs %v", err, tt.wantErrIs)
				}
				return
			}

			if tt.wantErrStr != "" {
				if err == nil {
					t.Fatalf("GetMyTicket() expected error containing %q, got nil", tt.wantErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetMyTicket() unexpected error: %v", err)
			}

			if ticket.QueueNumber != tt.wantQueueNum {
				t.Errorf("GetMyTicket() QueueNumber = %s, want %s", ticket.QueueNumber, tt.wantQueueNum)
			}
			if tt.wantWaitMin == nil {
				if ticket.EstimatedWaitTimeMinutes != nil {
					t.Errorf("GetMyTicket() EstimatedWaitTimeMinutes = %v, want nil", *ticket.EstimatedWaitTimeMinutes)
				}
			} else {
				if ticket.EstimatedWaitTimeMinutes == nil || *ticket.EstimatedWaitTimeMinutes != *tt.wantWaitMin {
					t.Errorf("GetMyTicket() EstimatedWaitTimeMinutes = %v, want %v", ticket.EstimatedWaitTimeMinutes, *tt.wantWaitMin)
				}
			}
			if ticket.Notice != tt.wantNotice {
				t.Errorf("GetMyTicket() Notice = %q, want %q", ticket.Notice, tt.wantNotice)
			}
			if ticket.PositionInQueue != tt.wantPos {
				t.Errorf("GetMyTicket() PositionInQueue = %d, want %d", ticket.PositionInQueue, tt.wantPos)
			}
			if ticket.AheadCount != tt.wantAhead {
				t.Errorf("GetMyTicket() AheadCount = %d, want %d", ticket.AheadCount, tt.wantAhead)
			}
		})
	}
}

func TestQueueUseCase_GetQueueStatus(t *testing.T) {
	errDB := errors.New("database error")

	tests := []struct {
		name           string
		setupMocks     func(q *mockQueueRepoPort, d *mockDoctorRepoPort)
		wantErrStr     string
		wantWaiting    int
		wantNotice     string
		wantQueueCount int
	}{
		{
			name: "Database error on GetAllDoctorsWithSessions",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get doctors with sessions",
		},
		{
			name: "Database error on GetActiveDoctors",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get active doctors",
		},
		{
			name: "Database error on GetWaitingTickets",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return nil, errDB
				}
			},
			wantErrStr: "get waiting tickets",
		},
		{
			name: "Success with active online doctors and waiting tickets",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{
						{ID: 1, Name: "Doctor A", AvgConsultationTimeMinutes: 3, IsOnline: true, Status: domain.DoctorStatusAvailable},
					}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{
						{ID: 1, Name: "Doctor A", AvgConsultationTime: 3, IsOnline: true},
					}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{
						{QueueNumber: "A-01", PatientName: "Alice"},
						{QueueNumber: "A-02", PatientName: "Bob"},
					}, nil
				}
			},
			wantWaiting:    2,
			wantNotice:     "",
			wantQueueCount: 2,
		},
		{
			name: "Success with all doctors offline",
			setupMocks: func(q *mockQueueRepoPort, d *mockDoctorRepoPort) {
				d.getAllDoctorsWithSessionsFunc = func(ctx context.Context) ([]domain.DoctorAvailability, error) {
					return []domain.DoctorAvailability{
						{ID: 1, Name: "Doctor A", AvgConsultationTimeMinutes: 3, IsOnline: false, Status: domain.DoctorStatusOffline},
					}, nil
				}
				d.getActiveDoctorsFunc = func(ctx context.Context) ([]*domain.Doctor, error) {
					return []*domain.Doctor{}, nil
				}
				q.getWaitingTicketsFunc = func(ctx context.Context) ([]*domain.QueueTicket, error) {
					return []*domain.QueueTicket{
						{QueueNumber: "A-01", PatientName: "Alice"},
					}, nil
				}
			},
			wantWaiting:    1,
			wantNotice:     "Estimated wait time is currently unavailable because all doctors are offline / on break. Calculation will activate once a doctor starts duty.",
			wantQueueCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := &mockQueueRepoPort{}
			mockDoc := &mockDoctorRepoPort{}
			tt.setupMocks(mockQueue, mockDoc)

			uc := NewQueueUseCase(mockQueue, mockDoc, nil)
			status, err := uc.GetQueueStatus(context.Background())

			if tt.wantErrStr != "" {
				if err == nil {
					t.Fatalf("GetQueueStatus() expected error containing %q, got nil", tt.wantErrStr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetQueueStatus() unexpected error: %v", err)
			}

			if status.TotalWaiting != tt.wantWaiting {
				t.Errorf("GetQueueStatus() TotalWaiting = %d, want %d", status.TotalWaiting, tt.wantWaiting)
			}
			if status.Notice != tt.wantNotice {
				t.Errorf("GetQueueStatus() Notice = %q, want %q", status.Notice, tt.wantNotice)
			}
			if len(status.QueueList) != tt.wantQueueCount {
				t.Errorf("GetQueueStatus() len(QueueList) = %d, want %d", len(status.QueueList), tt.wantQueueCount)
			}
		})
	}
}
