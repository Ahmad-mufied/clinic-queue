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

	"github.com/nats-io/nats.go"
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
			name:  "Handle QUEUE_JOINED without patient_name (Walk-in)",
			event: "QUEUE_JOINED",
			data:  map[string]any{},
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Walk-in Patient",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_CALLED with doctor_id only",
			event: "TICKET_CALLED",
			data: map[string]any{
				"doctor_id": float64(3),
			},
			expectedAction: "CONSULTATION_STARTED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 3",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_CALLED without doctor info",
			event: "TICKET_CALLED",
			data:  map[string]any{},
			expectedAction: "CONSULTATION_STARTED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Attending Doctor",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_FINISHED with doctor_id only",
			event: "TICKET_FINISHED",
			data: map[string]any{
				"doctor_id": float64(4),
			},
			expectedAction: "CONSULTATION_FINISHED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 4",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_FINISHED without doctor info",
			event: "TICKET_FINISHED",
			data:  map[string]any{},
			expectedAction: "CONSULTATION_FINISHED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Attending Doctor",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_STATUS_CHANGED with doctor_id only",
			event: "DOCTOR_STATUS_CHANGED",
			data: map[string]any{
				"doctor_id": float64(5),
			},
			expectedAction: "DOCTOR_STATUS_CHANGED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 5",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_STATUS_CHANGED without doctor info",
			event: "DOCTOR_STATUS_CHANGED",
			data:  map[string]any{},
			expectedAction: "DOCTOR_STATUS_CHANGED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Attending Doctor",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_LOGIN with username only and default role",
			event: "AUTH_LOGIN",
			data: map[string]any{
				"username": "user_only",
			},
			expectedAction: "AUTH_LOGIN",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "user_only",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_LOGIN without name or username",
			event: "AUTH_LOGIN",
			data:  map[string]any{},
			expectedAction: "AUTH_LOGIN",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "User",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_REGISTER with username only",
			event: "AUTH_REGISTER",
			data: map[string]any{
				"username": "reg_user",
			},
			expectedAction: "AUTH_REGISTER",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "reg_user",
			expectRecord:   true,
		},
		{
			name:  "Handle AUTH_REGISTER without name or username",
			event: "AUTH_REGISTER",
			data:  map[string]any{},
			expectedAction: "AUTH_REGISTER",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "New Patient",
			expectRecord:   true,
		},
		{
			name:  "Handle QUEUE_JOINED with nil/empty Data",
			event: "QUEUE_JOINED",
			data:  nil,
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Walk-in Patient",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_CONFIG_UPDATED with admin_id",
			event: "DOCTOR_CONFIG_UPDATED",
			data: map[string]any{
				"admin_id":  "01919df4-8e3b-7412-a1f9-90b567c9e203",
				"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101",
				"avg_time":  float64(5),
			},
			expectedAction: "DOCTOR_CONFIG_UPDATED",
			expectedRole:   string(domain.RoleAdmin),
			expectedActor:  "Clinic Administrator",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_CALLED with doctor_id as string",
			event: "TICKET_CALLED",
			data: map[string]any{
				"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101",
			},
			expectedAction: "CONSULTATION_STARTED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 01919df4-8e3b-7412-a1f9-90b567c9e101",
			expectRecord:   true,
		},
		{
			name:  "Handle TICKET_FINISHED with doctor_id as string",
			event: "TICKET_FINISHED",
			data: map[string]any{
				"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101",
			},
			expectedAction: "CONSULTATION_FINISHED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 01919df4-8e3b-7412-a1f9-90b567c9e101",
			expectRecord:   true,
		},
		{
			name:  "Handle DOCTOR_STATUS_CHANGED with doctor_id as string",
			event: "DOCTOR_STATUS_CHANGED",
			data: map[string]any{
				"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101",
			},
			expectedAction: "DOCTOR_STATUS_CHANGED",
			expectedRole:   string(domain.RoleDoctor),
			expectedActor:  "Dr. Doctor 01919df4-8e3b-7412-a1f9-90b567c9e101",
			expectRecord:   true,
		},
		{
			name:  "Handle QUEUE_JOINED with invalid/empty user_id types",
			event: "QUEUE_JOINED",
			data: map[string]any{
				"user_id":      "   ",
				"patient_name": "Walk-in Patient",
			},
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Walk-in Patient",
			expectRecord:   true,
		},
		{
			name:  "Handle QUEUE_JOINED with negative float and bool user_id",
			event: "QUEUE_JOINED",
			data: map[string]any{
				"user_id":      float64(-5),
				"patient_name": "Walk-in Patient",
			},
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Walk-in Patient",
			expectRecord:   true,
		},
		{
			name:  "Handle QUEUE_JOINED with bool user_id",
			event: "QUEUE_JOINED",
			data: map[string]any{
				"user_id":      true,
				"patient_name": "Walk-in Patient",
			},
			expectedAction: "QUEUE_JOINED",
			expectedRole:   string(domain.RolePatient),
			expectedActor:  "Walk-in Patient",
			expectRecord:   true,
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
					return &domain.AuditLog{ID: "01919df4-8e3b-7412-a1f9-90b567c9e501"}, tt.recordLogErr
				},
			}

			w := worker.NewAuditWorker(mockUC, &mockUserRepo{})

			var msgBytes []byte
			if tt.rawJSON != nil {
				msgBytes = tt.rawJSON
			} else {
				var dataBytes []byte
				if tt.data != nil {
					dataBytes, _ = json.Marshal(tt.data)
				}
				envelope := worker.EventEnvelope{
					Type:      tt.event,
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

func TestAuditWorker_StartSubscribing(t *testing.T) {
	mockUC := &mockAuditUseCase{}
	w := worker.NewAuditWorker(mockUC, &mockUserRepo{})

	// Test nil connection
	_, err := w.StartSubscribing(context.Background(), nil, "clinic.events.>")
	if err == nil {
		t.Error("expected error for nil nats conn, got nil")
	}

	// Test with live NATS connection if running
	nc, err := nats.Connect("nats://localhost:4222", nats.Timeout(500*time.Millisecond))
	if err == nil {
		sub, err := w.StartSubscribing(context.Background(), nc, "test.clinic.events.worker")
		if err != nil {
			t.Fatalf("unexpected subscription error: %v", err)
		}

		// Publish message to trigger subscription handler callback
		env := worker.EventEnvelope{
			Type:      "QUEUE_JOINED",
			Data:      []byte(`{"patient_name":"Sub Test"}`),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		envBytes, _ := json.Marshal(env)
		_ = nc.Publish("test.clinic.events.worker", envBytes)
		_ = nc.Flush()
		time.Sleep(50 * time.Millisecond)

		if sub != nil {
			_ = sub.Unsubscribe()
		}

		// Test subscribe on closed connection returns error
		nc.Close()
		_, err = w.StartSubscribing(context.Background(), nc, "test.closed.>")
		if err == nil {
			t.Error("expected error when subscribing on closed nats conn, got nil")
		}
	}
}
