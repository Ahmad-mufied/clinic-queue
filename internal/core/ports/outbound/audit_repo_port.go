package outbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// AuditRepositoryPort defines the driven/outbound SPI interface for audit log storage in PostgreSQL 18.
type AuditRepositoryPort interface {
	// InsertLog persists a new immutable audit log entry in the database.
	InsertLog(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error)

	// QueryLogs retrieves paginated audit logs based on the provided filter parameters.
	QueryLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error)
}
