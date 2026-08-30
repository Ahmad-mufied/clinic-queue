package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/nats-io/nats.go"
)

// AuditWorker subscribes to NATS JetStream domain events and writes immutable records to audit_logs.
type AuditWorker struct {
	auditUseCase inbound.AuditUseCase
	userRepo     outbound.UserRepositoryPort
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
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	Timestamp string          `json:"timestamp"`
}

// StartSubscribing registers an asynchronous NATS subscriber on the given subject (e.g. "clinic.events.>").
func (w *AuditWorker) StartSubscribing(ctx context.Context, nc *nats.Conn, subject string) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		w.HandleEventMessage(ctx, msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}

	return sub, nil
}

// HandleEventMessage processes incoming serialized NATS events and inserts audit log entries.
func (w *AuditWorker) HandleEventMessage(ctx context.Context, data []byte) {
	var envelope EventEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return
	}

	// Skip recursive or purely UI stream-refresh events
	if envelope.Event == "AUDIT_LOG_CREATED" || envelope.Event == "QUEUE_UPDATED" {
		return
	}

	var rawMap map[string]any
	_ = json.Unmarshal(envelope.Data, &rawMap)
	if rawMap == nil {
		rawMap = make(map[string]any)
	}

	dto := inbound.RecordAuditLogDTO{
		IPAddress: "127.0.0.1",
		Details:   rawMap,
	}

	switch envelope.Event {
	case "QUEUE_JOINED":
		dto.Action = "QUEUE_JOINED"
		dto.Role = string(domain.RolePatient)
		if name, ok := rawMap["patient_name"].(string); ok && name != "" {
			dto.ActorName = name
		} else {
			dto.ActorName = "Walk-in Patient"
		}
		if uid, ok := rawMap["user_id"].(float64); ok && uid > 0 {
			intUID := int(uid)
			dto.UserID = &intUID
		}

	case "TICKET_CALLED":
		dto.Action = "CONSULTATION_STARTED"
		dto.Role = string(domain.RoleDoctor)
		if docName, ok := rawMap["doctor_name"].(string); ok && docName != "" {
			dto.ActorName = docName
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}

	case "TICKET_FINISHED":
		dto.Action = "CONSULTATION_FINISHED"
		dto.Role = string(domain.RoleDoctor)
		if docName, ok := rawMap["doctor_name"].(string); ok && docName != "" {
			dto.ActorName = docName
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}

	case "DOCTOR_STATUS_CHANGED":
		dto.Action = "DOCTOR_STATUS_CHANGED"
		dto.Role = string(domain.RoleDoctor)
		if name, ok := rawMap["name"].(string); ok && name != "" {
			dto.ActorName = name
		} else if docID, ok := rawMap["doctor_id"].(float64); ok {
			dto.ActorName = fmt.Sprintf("Dr. Doctor %d", int(docID))
		} else {
			dto.ActorName = "Attending Doctor"
		}

	case "DOCTOR_CONFIG_UPDATED":
		dto.Action = "DOCTOR_CONFIG_UPDATED"
		dto.Role = string(domain.RoleAdmin)
		dto.ActorName = "Clinic Administrator"

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
		if uid, ok := rawMap["user_id"].(float64); ok && uid > 0 {
			intUID := int(uid)
			dto.UserID = &intUID
		}

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
		if uid, ok := rawMap["user_id"].(float64); ok && uid > 0 {
			intUID := int(uid)
			dto.UserID = &intUID
		}

	default:
		return
	}

	if _, err := w.auditUseCase.RecordLog(ctx, dto); err != nil {
		log.Printf("Failed to record audit log for %s: %v", envelope.Event, err)
	}
}
