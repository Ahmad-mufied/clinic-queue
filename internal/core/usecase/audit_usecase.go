package usecase

import (
	"context"
	"fmt"
	"strings"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"
)

// AuditUseCase implements inbound.AuditUseCase for forensic activity logging and audit queries.
type AuditUseCase struct {
	auditRepo outbound.AuditRepositoryPort
	eventPub  outbound.EventPublisherPort
}

// NewAuditUseCase constructs a new AuditUseCase instance.
func NewAuditUseCase(
	auditRepo outbound.AuditRepositoryPort,
	eventPub outbound.EventPublisherPort,
) *AuditUseCase {
	return &AuditUseCase{
		auditRepo: auditRepo,
		eventPub:  eventPub,
	}
}

var _ inbound.AuditUseCase = (*AuditUseCase)(nil)

// RecordLog records an immutable activity log entry and broadcasts the event via real-time streaming.
func (u *AuditUseCase) RecordLog(ctx context.Context, dto inbound.RecordAuditLogDTO) (*domain.AuditLog, error) {
	if strings.TrimSpace(dto.Action) == "" {
		return nil, domain.ErrInvalidAction
	}

	entry := &domain.AuditLog{
		UserID:    dto.UserID,
		ActorName: dto.ActorName,
		Role:      dto.Role,
		Action:    dto.Action,
		Details:   dto.Details,
		IPAddress: dto.IPAddress,
	}
	entry.Normalize()

	savedLog, err := u.auditRepo.InsertLog(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	if u.eventPub != nil {
		_ = u.eventPub.PublishEvent(ctx, "AUDIT_LOG_CREATED", savedLog)
	}

	return savedLog, nil
}

// GetAuditLogs retrieves a paginated and filtered list of audit log records.
func (u *AuditUseCase) GetAuditLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	filter.NormalizePagination()

	result, err := u.auditRepo.QueryLogs(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}

	if result == nil {
		result = &domain.PaginatedAuditLogs{
			Page:         filter.Page,
			Limit:        filter.Limit,
			TotalRecords: 0,
			Logs:         []domain.AuditLog{},
		}
	}

	if result.Logs == nil {
		result.Logs = []domain.AuditLog{}
	}

	return result, nil
}
