package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

type mockAuditRepoPort struct {
	insertLogFunc func(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error)
	queryLogsFunc func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error)
}

func (m *mockAuditRepoPort) InsertLog(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
	if m != nil && m.insertLogFunc != nil {
		return m.insertLogFunc(ctx, log)
	}
	return nil, nil
}

func (m *mockAuditRepoPort) QueryLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	if m != nil && m.queryLogsFunc != nil {
		return m.queryLogsFunc(ctx, filter)
	}
	return nil, nil
}

type mockAuditEventPubPort struct {
	publishEventFunc func(ctx context.Context, eventType string, payload any) error
	closeFunc        func() error
}

func (m *mockAuditEventPubPort) PublishEvent(ctx context.Context, eventType string, payload any) error {
	if m != nil && m.publishEventFunc != nil {
		return m.publishEventFunc(ctx, eventType, payload)
	}
	return nil
}

func (m *mockAuditEventPubPort) Close() error {
	if m != nil && m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

var (
	_ outbound.AuditRepositoryPort = (*mockAuditRepoPort)(nil)
	_ outbound.EventPublisherPort  = (*mockAuditEventPubPort)(nil)
)

func TestAuditUseCase_RecordLog(t *testing.T) {
	userID := 1
	now := time.Now().UTC()

	tests := []struct {
		name         string
		dto          inbound.RecordAuditLogDTO
		auditRepo    *mockAuditRepoPort
		eventPub     *mockAuditEventPubPort
		wantErr      error
		wantAction   string
		wantActor    string
		wantRole     string
		wantIP       string
		wantPubEvent string
	}{
		{
			name: "empty action returns ErrInvalidAction",
			dto: inbound.RecordAuditLogDTO{
				Action: "  ",
			},
			auditRepo: nil,
			wantErr:   domain.ErrInvalidAction,
		},
		{
			name: "repository insert failure returns wrapped error",
			dto: inbound.RecordAuditLogDTO{
				Action:    domain.ActionAuthLogin,
				ActorName: "dr_smith",
				Role:      "doctor",
				IPAddress: "127.0.0.1",
			},
			auditRepo: &mockAuditRepoPort{
				insertLogFunc: func(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
					return nil, errors.New("db insert failure")
				},
			},
			wantErr: errors.New("db insert failure"),
		},
		{
			name: "successful insert with normalization and event publishing",
			dto: inbound.RecordAuditLogDTO{
				UserID:    &userID,
				Action:    domain.ActionQueueJoined,
				ActorName: "", // will be normalized to default
				Role:      "", // will be normalized to default
				IPAddress: "", // will be normalized to default
				Details:   nil,
			},
			auditRepo: &mockAuditRepoPort{
				insertLogFunc: func(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
					return &domain.AuditLog{
						ID:        100,
						UserID:    log.UserID,
						ActorName: log.ActorName,
						Role:      log.Role,
						Action:    log.Action,
						Details:   log.Details,
						IPAddress: log.IPAddress,
						CreatedAt: now,
					}, nil
				},
			},
			eventPub: &mockAuditEventPubPort{
				publishEventFunc: func(ctx context.Context, eventType string, payload any) error {
					if eventType != "AUDIT_LOG_CREATED" {
						t.Errorf("expected event type AUDIT_LOG_CREATED, got %s", eventType)
					}
					return nil
				},
			},
			wantErr:      nil,
			wantAction:   domain.ActionQueueJoined,
			wantActor:    domain.DefaultAnonymousActor,
			wantRole:     domain.DefaultFallbackRole,
			wantIP:       domain.DefaultFallbackIP,
			wantPubEvent: "AUDIT_LOG_CREATED",
		},
		{
			name: "successful insert with nil event publisher",
			dto: inbound.RecordAuditLogDTO{
				Action:    domain.ActionDoctorShiftStarted,
				ActorName: "Dr. Alice",
				Role:      "doctor",
				IPAddress: "192.168.1.50",
				Details:   map[string]any{"doctor_id": 2},
			},
			auditRepo: &mockAuditRepoPort{
				insertLogFunc: func(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
					return &domain.AuditLog{
						ID:        101,
						ActorName: log.ActorName,
						Role:      log.Role,
						Action:    log.Action,
						Details:   log.Details,
						IPAddress: log.IPAddress,
						CreatedAt: now,
					}, nil
				},
			},
			eventPub:   nil,
			wantErr:    nil,
			wantAction: domain.ActionDoctorShiftStarted,
			wantActor:  "Dr. Alice",
			wantRole:   "doctor",
			wantIP:     "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAuditUseCase(tt.auditRepo, tt.eventPub)
			res, err := uc.RecordLog(context.Background(), tt.dto)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Action != tt.wantAction {
				t.Errorf("expected Action %s, got %s", tt.wantAction, res.Action)
			}
			if res.ActorName != tt.wantActor {
				t.Errorf("expected ActorName %s, got %s", tt.wantActor, res.ActorName)
			}
			if res.Role != tt.wantRole {
				t.Errorf("expected Role %s, got %s", tt.wantRole, res.Role)
			}
			if res.IPAddress != tt.wantIP {
				t.Errorf("expected IPAddress %s, got %s", tt.wantIP, res.IPAddress)
			}
		})
	}
}

func TestAuditUseCase_GetAuditLogs(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name            string
		filter          domain.AuditLogFilter
		auditRepo       *mockAuditRepoPort
		wantErr         error
		wantPage        int
		wantLimit       int
		wantTotal       int
		wantLogsCount   int
	}{
		{
			name: "repository query failure returns wrapped error",
			filter: domain.AuditLogFilter{
				Page:  1,
				Limit: 20,
			},
			auditRepo: &mockAuditRepoPort{
				queryLogsFunc: func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, errors.New("db query failure")
				},
			},
			wantErr: errors.New("db query failure"),
		},
		{
			name: "successful query with results",
			filter: domain.AuditLogFilter{
				Action: domain.ActionAuthLogin,
				Role:   "doctor",
				Page:   1,
				Limit:  10,
			},
			auditRepo: &mockAuditRepoPort{
				queryLogsFunc: func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return &domain.PaginatedAuditLogs{
						Page:         filter.Page,
						Limit:        filter.Limit,
						TotalRecords: 1,
						Logs: []domain.AuditLog{
							{
								ID:        1,
								ActorName: "dr_smith",
								Role:      "doctor",
								Action:    domain.ActionAuthLogin,
								CreatedAt: now,
							},
						},
					}, nil
				},
			},
			wantErr:       nil,
			wantPage:      1,
			wantLimit:     10,
			wantTotal:     1,
			wantLogsCount: 1,
		},
		{
			name: "successful query when repo returns nil result (zero state)",
			filter: domain.AuditLogFilter{
				Page:  0,
				Limit: 0,
			},
			auditRepo: &mockAuditRepoPort{
				queryLogsFunc: func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return nil, nil
				},
			},
			wantErr:       nil,
			wantPage:      domain.DefaultPage,
			wantLimit:     domain.DefaultLimit,
			wantTotal:     0,
			wantLogsCount: 0,
		},
		{
			name: "successful query when repo returns result with nil Logs slice",
			filter: domain.AuditLogFilter{
				Page:  2,
				Limit: 15,
			},
			auditRepo: &mockAuditRepoPort{
				queryLogsFunc: func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
					return &domain.PaginatedAuditLogs{
						Page:         filter.Page,
						Limit:        filter.Limit,
						TotalRecords: 0,
						Logs:         nil,
					}, nil
				},
			},
			wantErr:       nil,
			wantPage:      2,
			wantLimit:     15,
			wantTotal:     0,
			wantLogsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAuditUseCase(tt.auditRepo, nil)
			res, err := uc.GetAuditLogs(context.Background(), tt.filter)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.Page != tt.wantPage {
				t.Errorf("expected Page %d, got %d", tt.wantPage, res.Page)
			}
			if res.Limit != tt.wantLimit {
				t.Errorf("expected Limit %d, got %d", tt.wantLimit, res.Limit)
			}
			if res.TotalRecords != tt.wantTotal {
				t.Errorf("expected TotalRecords %d, got %d", tt.wantTotal, res.TotalRecords)
			}
			if len(res.Logs) != tt.wantLogsCount {
				t.Errorf("expected Logs count %d, got %d", tt.wantLogsCount, len(res.Logs))
			}
		})
	}
}

func TestAuditMockDefaults(t *testing.T) {
	// Call default branches of mocks when functions are nil to ensure 100% test coverage
	var repoMock mockAuditRepoPort
	l, err := repoMock.InsertLog(context.Background(), nil)
	if l != nil || err != nil {
		t.Errorf("expected nil, nil")
	}
	p, err := repoMock.QueryLogs(context.Background(), domain.AuditLogFilter{})
	if p != nil || err != nil {
		t.Errorf("expected nil, nil")
	}

	var eventMock mockAuditEventPubPort
	err = eventMock.PublishEvent(context.Background(), "EVT", nil)
	if err != nil {
		t.Errorf("expected nil")
	}
	err = eventMock.Close()
	if err != nil {
		t.Errorf("expected nil")
	}
}
