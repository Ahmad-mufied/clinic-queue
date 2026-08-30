package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"clinic-queue/internal/adapters/inbound/worker"
	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

type mockAuditUseCase struct {
	recordLogFn   func(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error)
	getAuditLogsFn func(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error)
}

func (m *mockAuditUseCase) RecordLog(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error) {
	if m.recordLogFn != nil {
		return m.recordLogFn(ctx, dto)
	}
	return &domain.AuditLog{}, nil
}

func (m *mockAuditUseCase) GetAuditLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	if m.getAuditLogsFn != nil {
		return m.getAuditLogsFn(ctx, filter)
	}
	return &domain.PaginatedAuditLogs{}, nil
}

type mockUserRepo struct {
	outbound.UserRepositoryPort
}

func TestAuditWorker_HandleEventMessage(t *testing.T) {
	tests := []struct {
		name          string
		event         string
		data          map[string]any
		rawJSON       []byte
		recordLogErr  error
		expectedAction string
		expectedRole  string
		expectedActor string
		expectRecord  bool
	}{
		{
			name:  "Handle QUEUE_JOINED",
			event: "QUEUE_JOINED",
			data: map[string]any{
				"patient_name": "Alice Smith",
				"user_id":      float64(10),
				"queue_number": "A-01",
			},
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Alice Smith",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_CALLED",
			event: "TICKET_CALLED",
			data: map[string]any{
				"doctor_name":  "Dr. Sarah Adams",
				"doctor_id":    float64(1),
				"patient_name": "Alice Smith",
				"ticket_id":    float64(5),
			},
			expectedAction: "CONSULTATION_STARTED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Sarah Adams",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_FINISHED",
			event: "TICKET_FINISHED",
			data: map[string]any{
				"doctor_name":            "Dr. Michael Chen",
				"doctor_id":              float64(2),
				"patient_name":           "Lucas Smith",
				"actual_duration_minutes": float64(4.2),
			},
			expectedAction: "CONSULTATION_FINISHED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Michael Chen",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_STATUS_CHANGED",
			event: "DOCTOR_STATUS_CHANGED",
			data: map[string]any{
				"name":      "Dr. Sarah Adams",
				"doctor_id": float64(1),
				"is_online": true,
				"status":    "AVAILABLE",
			},
			expectedAction: "DOCTOR_STATUS_CHANGED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Sarah Adams",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_CONFIG_UPDATED",
			event: "DOCTOR_CONFIG_UPDATED",
			data: map[string]any{
				"doctor_id": float64(1),
				"avg_time":  float64(5),
			},
			expectedAction: "DOCTOR_CONFIG_UPDATED",
			expectedRole:   string(domain.RoleAdmin),
			expectedActor:  "Clinic Administrator",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_LOGIN",
			event: "AUTH_LOGIN",
			data: map[string]any{
				"user_id":  float64(3),
				"username": "patient_lucas",
				"name":     "Lucas Smith",
				"role":     "patient",
			},
			expectedAction: "AUTH_LOGIN",
			expectedRole:   "patient",
			expectedActor:  "Lucas Smith",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_REGISTER",
			event: "AUTH_REGISTER",
			data: map[string]any{
				"user_id":  float64(4),
				"username": "new_patient",
				"name":     "New Patient",
				"role":     "patient",
			},
			expectedAction: "AUTH_REGISTER",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "New Patient",
			expectRecord:   true,
		},
		{
			name:         "Skip AUDIT_LOG_CREATED",
			event:        "AUDIT_LOG_CREATED",
			data:         map[string]any{"id": 1},
			expectRecord: false,
		},
		{
			name:         "Skip QUEUE_UPDATED",
			event:        "QUEUE_UPDATED",
			data:         map[string]any{"status": "ok"},
			expectRecord: false,
		},
		{
			name:         "Skip Unknown event",
			event:        "UNKNOWN_EVENT",
			data:         map[string]any{"foo": "bar"},
			expectRecord: false,
		},
		{
			name:         "Invalid JSON payload",
			rawJSON:      []byte("invalid json"),
			expectRecord: false,
		},
		{
			name:  "Handle Error in RecordLog gracefully",
			event: "QUEUE_JOINED",
			data: map[string]any{
				"patient_name": "Error Patient",
			},
			recordLogErr: errors.New("database insert failed"),
			expectRecord: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorded := false
			mockUC := &mockAuditUseCase{
				recordLogFn: func(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error) {
					recorded = true
					if tt.expectedAction != "" && dto.Action != tt.expectedAction {
						t.Errorf("expected action %q, got %q", tt.expectedAction, dto.Action)
					}
					if tt.expectedRole != "" && dto.Role != tt.expectedRole {
						t.Errorf("expected role %q, got %q", tt.expectedRole, dto.Role)
					}
					if tt.expectedActor != "" && dto.ActorName != tt.expectedActor {
						t.Errorf("expected actor %q, got %q", tt.expectedActor, dto.ActorName)
					}
					return &domain.AuditLog{ID: 1}, tt.recordLogErr
				},
			}

			w := worker.NewAuditWorker(mockUC, &mockUserRepo{})

			var msgBytes []byte
			if tt.rawJSON != nil {
				msgBytes = tt.rawJSON
			} else {
				dataBytes, _ := json.Marshal(tt.data)
				envelope := worker.EventEnvelope{
					Event:     tt.event,
					Data:      dataBytes,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				}
				msgBytes, _ = json.Marshal(envelope)
			}

			w.HandleEventMessage(context.Background(), msgBytes)

			if recorded != tt.expectRecord {
				t.Errorf("expected record=%v, got %v", tt.expectRecord, recorded)
			}
		})
	}
}

func TestAuditWorker_StartSubscribing_NilNC(t *testing.T) {
	w := worker.NewAuditWorker(&mockAuditUseCase{}, &mockUserRepo{})
	_, err := w.StartSubscribing(context.Background(), nil, "clinic.events.>")
	if err == nil {
		t.Error("expected error for nil nats conn, got nil")
	}
}
