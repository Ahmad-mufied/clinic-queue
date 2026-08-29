package domain

import (
	"strings"
	"time"
)

// Standard Audit Action Constants
const (
	ActionAuthLogin            = "AUTH_LOGIN"
	ActionAuthLogout           = "AUTH_LOGOUT"
	ActionAuthRegister         = "AUTH_REGISTER"
	ActionDoctorShiftStarted   = "DOCTOR_SHIFT_STARTED"
	ActionDoctorShiftEnded     = "DOCTOR_SHIFT_ENDED"
	ActionQueueJoined          = "QUEUE_JOINED"
	ActionConsultationStarted  = "CONSULTATION_STARTED"
	ActionConsultationFinished = "CONSULTATION_FINISHED"
	ActionQueueCancelled       = "QUEUE_CANCELLED"
	ActionDoctorConfigUpdated  = "DOCTOR_CONFIG_UPDATED"
)

// Default fallback values for activity logging and pagination
const (
	DefaultAnonymousActor = "Anonymous / System"
	DefaultFallbackRole   = "public"
	DefaultFallbackIP     = "127.0.0.1"
	DefaultPage           = 1
	DefaultLimit          = 20
	MaxLimit              = 100
)

// AuditLog represents an immutable forensic and operational activity record.
type AuditLog struct {
	ID        int            `json:"id"`
	UserID    *int           `json:"user_id,omitempty"`
	ActorName string         `json:"actor_name"`
	Role      string         `json:"role"`
	Action    string         `json:"action"`
	Details   map[string]any `json:"details"`
	IPAddress string         `json:"ip_address,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// AuditLogFilter defines search, filter, and pagination parameters for querying audit records.
type AuditLogFilter struct {
	Action string `json:"action"`
	Role   string `json:"role"`
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
}

// PaginatedAuditLogs represents the paginated result set of audit logs.
type PaginatedAuditLogs struct {
	Page         int        `json:"page"`
	Limit        int        `json:"limit"`
	TotalRecords int        `json:"total_records"`
	Logs         []AuditLog `json:"logs"`
}

// Validate validates the core domain fields of an AuditLog.
func (a *AuditLog) Validate() error {
	if a == nil {
		return ErrInvalidInput
	}
	if strings.TrimSpace(a.Action) == "" {
		return ErrInvalidAction
	}
	if strings.TrimSpace(a.ActorName) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(a.Role) == "" {
		return ErrInvalidRole
	}
	return nil
}

// Normalize sets safe default values for empty fields in an AuditLog before insertion.
func (a *AuditLog) Normalize() {
	if a == nil {
		return
	}
	if strings.TrimSpace(a.ActorName) == "" {
		a.ActorName = DefaultAnonymousActor
	}
	if strings.TrimSpace(a.Role) == "" {
		a.Role = DefaultFallbackRole
	}
	if strings.TrimSpace(a.IPAddress) == "" {
		a.IPAddress = DefaultFallbackIP
	}
	if a.Details == nil {
		a.Details = make(map[string]any)
	}
}

// NormalizePagination ensures page and limit values are valid positive integers capped at max bounds.
func (f *AuditLogFilter) NormalizePagination() {
	if f == nil {
		return
	}
	if f.Page <= 0 {
		f.Page = DefaultPage
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
}

// Offset returns the SQL OFFSET value based on current page and limit.
func (f *AuditLogFilter) Offset() int {
	if f == nil || f.Page <= 1 {
		return 0
	}
	return (f.Page - 1) * f.Limit
}
