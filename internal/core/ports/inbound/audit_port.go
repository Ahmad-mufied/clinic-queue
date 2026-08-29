package inbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// RecordAuditLogDTO represents the input payload for creating a new audit trail record.
type RecordAuditLogDTO struct {
	UserID    *int           `json:"user_id,omitempty"`
	ActorName string         `json:"actor_name"`
	Role      string         `json:"role"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details"`
	IPAddress string         `json:"ip_address,omitempty"`
}

// AuditUseCase defines the driving/inbound port for recording activity logs and querying the audit trail.
type AuditUseCase interface {
	// RecordLog records an immutable activity log entry and broadcasts the event via real-time streaming.
	RecordLog(ctx context.Context, dto RecordAuditLogDTO) (*domain.AuditLog, error)

	// GetAuditLogs retrieves a paginated and filtered list of audit log records.
	GetAuditLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error)
}
