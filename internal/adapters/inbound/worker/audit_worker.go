package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/nats-io/nats.go"
)

// AuditWorker subscribes to NATS JetStream domain events and writes immutable records to audit_logs.
type AuditWorker struct {
	auditUseCase inbound.AuditUseCase
	userRepo     outbound.UserRepositoryPort
	mu           sync.Mutex
	inFlight     int64
	drainCh      chan struct{}
}

// NewAuditWorker constructs a new AuditWorker instance.
func NewAuditWorker(auditUseCase inbound.AuditUseCase, userRepo outbound.UserRepositoryPort) *AuditWorker {
	return &AuditWorker{
		auditUseCase: auditUseCase,
		userRepo:     userRepo,
	}
}

// EventEnvelope represents the standard event message payload published across the platform.
type EventEnvelope struct {
	Type      string                `json:"type"`
	Data      json.RawMessage       `json:"data"`
	Timestamp string                `json:"timestamp"`
	Metadata  domain.ClientMetadata `json:"metadata,omitempty"`
}

// StartSubscribing registers an asynchronous NATS subscriber on the given subject (e.g. "clinic.events.>").
func (w *AuditWorker) StartSubscribing(ctx context.Context, nc *nats.Conn, subject string) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		w.mu.Lock()
		w.inFlight++
		w.mu.Unlock()

		defer func() {
			w.mu.Lock()
			w.inFlight--
			if w.inFlight == 0 && w.drainCh != nil {
				close(w.drainCh)
				w.drainCh = nil
			}
			w.mu.Unlock()
		}()

		w.HandleEventMessage(ctx, msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	return sub, nil
}

// Wait blocks until all in-flight event processing goroutines have completed, or until ctx expires.
func (w *AuditWorker) Wait(ctx context.Context) error {
	w.mu.Lock()
	if w.inFlight == 0 {
		w.mu.Unlock()
		return nil
	}
	if w.drainCh == nil {
		w.drainCh = make(chan struct{})
	}
	ch := w.drainCh
	w.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func extractID(m map[string]any, key string) *string {
	val, ok := m[key]
	if !ok || val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return &trimmed
		}
	case float64:
		if v > 0 {
			str := fmt.Sprintf("%.0f", v)
			return &str
		}
	}
	return nil
}

// HandleEventMessage processes incoming serialized NATS events and inserts audit log entries.
func (w *AuditWorker) HandleEventMessage(ctx context.Context, data []byte) {
	var envelope EventEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return
	}

	// Skip recursive or purely UI stream-refresh events
	if envelope.Type == "AUDIT_LOG_CREATED" || envelope.Type == "QUEUE_UPDATED" {
		return
	}

	var rawMap map[string]any
	_ = json.Unmarshal(envelope.Data, &rawMap)
	if rawMap == nil {
		rawMap = make(map[string]any)
	}

	ipAddr := strings.TrimSpace(envelope.Metadata.ClientIP)
	if ipAddr == "" {
		ipAddr = "127.0.0.1"
	}

	if ua := strings.TrimSpace(envelope.Metadata.UserAgent); ua != "" {
		rawMap["user_agent"] = ua
	}
	if reqID := strings.TrimSpace(envelope.Metadata.RequestID); reqID != "" {
		rawMap["request_id"] = reqID
	}
	if envelope.Metadata.ClientIP != "" {
		rawMap["client_ip"] = envelope.Metadata.ClientIP
	}

	dto := inbound.RecordAuditLogDTO{
		IPAddress: ipAddr,
		Details:   rawMap,
	}

	switch envelope.Type {
	case "QUEUE_JOINED":
		dto.Action = "QUEUE_JOINED"
		dto.Role = string(domain.RolePatient)
		if name, ok := rawMap["patient_name"].(string); ok && name != "" {
			dto.ActorName = name
		} else {
			dto.ActorName = "Walk-in Patient"
		}
		dto.UserID = extractID(rawMap, "user_id")

	case "TICKET_CALLED":
		dto.Action = "CONSULTATION_STARTED"
		dto.Role = string(domain.RoleDoctor)
		if docName, ok := rawMap["doctor_name"].(string); ok && docName != "" {
			dto.ActorName = docName
		} else if docID, ok := rawMap["doctor_id"].(string); ok && docID != "" {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %s", docID)
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}
		dto.UserID = extractID(rawMap, "user_id")

	case "TICKET_FINISHED":
		dto.Action = "CONSULTATION_FINISHED"
		dto.Role = string(domain.RoleDoctor)
		if docName, ok := rawMap["doctor_name"].(string); ok && docName != "" {
			dto.ActorName = docName
		} else if docID, ok := rawMap["doctor_id"].(string); ok && docID != "" {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %s", docID)
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}
		dto.UserID = extractID(rawMap, "user_id")

	case "DOCTOR_STATUS_CHANGED":
		dto.Action = "DOCTOR_STATUS_CHANGED"
		dto.Role = string(domain.RoleDoctor)
		if name, ok := rawMap["name"].(string); ok && name != "" {
			dto.ActorName = name
		} else if docID, ok := rawMap["doctor_id"].(string); ok && docID != "" {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %s", docID)
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}
		dto.UserID = extractID(rawMap, "user_id")

	case "DOCTOR_CONFIG_UPDATED":
		dto.Action = "DOCTOR_CONFIG_UPDATED"
		dto.Role = string(domain.RoleAdmin)
		dto.ActorName = "Clinic Administrator"
		if adminUID := extractID(rawMap, "admin_id"); adminUID != nil {
			dto.UserID = adminUID
		} else if uID := extractID(rawMap, "user_id"); uID != nil {
			dto.UserID = uID
		} else {
			adminID := "01919df4-8e3b-7412-a1f9-90b567c9e205"
			dto.UserID = &adminID
		}

	case "AUTH_LOGIN":
		dto.Action = "AUTH_LOGIN"
		if role, ok := rawMap["role"].(string); ok && role != "" {
			dto.Role = role
		} else {
			dto.Role = string(domain.RolePatient)
		}
		if name, ok := rawMap["name"].(string); ok && name != "" {
			dto.ActorName = name
		} else if uName, ok := rawMap["username"].(string); ok && uName != "" {
			dto.ActorName = uName
		} else {
			dto.ActorName = "User"
		}
		dto.UserID = extractID(rawMap, "user_id")

	case "AUTH_REGISTER":
		dto.Action = "AUTH_REGISTER"
		dto.Role = string(domain.RolePatient)
		if name, ok := rawMap["name"].(string); ok && name != "" {
			dto.ActorName = name
		} else if uName, ok := rawMap["username"].(string); ok && uName != "" {
			dto.ActorName = uName
		} else {
			dto.ActorName = "New Patient"
		}
		dto.UserID = extractID(rawMap, "user_id")

	default:
		return
	}

	if _, err := w.auditUseCase.RecordLog(ctx, dto); err != nil {
		log.Printf("Failed to record audit log for %s: %v", envelope.Type, err)
	}
}
