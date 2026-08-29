package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepo implements the outbound.AuditRepositoryPort interface using PostgreSQL 18.
type AuditRepo struct {
	pool *pgxpool.Pool
}

// NewAuditRepo constructs a new AuditRepo instance.
func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

var _ outbound.AuditRepositoryPort = (*AuditRepo)(nil)

// InsertLog persists a new immutable audit log entry into PostgreSQL.
func (r *AuditRepo) InsertLog(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
	if log.Details == nil {
		log.Details = make(map[string]any)
	}

	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return nil, fmt.Errorf("marshal audit details to json: %w", err)
	}

	query := `
		INSERT INTO audit_logs (user_id, actor_name, role, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`

	err = r.pool.QueryRow(
		ctx,
		query,
		log.UserID,
		log.ActorName,
		log.Role,
		log.Action,
		detailsJSON,
		log.IPAddress,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert audit log record: %w", err)
	}

	return log, nil
}

// QueryLogs retrieves paginated audit logs based on the provided filter parameters.
func (r *AuditRepo) QueryLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if strings.TrimSpace(filter.Action) != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Action))
		argIdx++
	}

	if strings.TrimSpace(filter.Role) != "" {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Role))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			id, user_id, actor_name, role, action, details, ip_address, created_at,
			COUNT(*) OVER() AS total_records
		FROM audit_logs
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, filter.Limit, filter.Offset())

	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	var totalRecords int

	for rows.Next() {
		var (
			item         domain.AuditLog
			detailsBytes []byte
			ipAddress    *string
			totRec       int
		)

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.ActorName,
			&item.Role,
			&item.Action,
			&detailsBytes,
			&ipAddress,
			&item.CreatedAt,
			&totRec,
		); err != nil {
			return nil, fmt.Errorf("scan audit log row: %w", err)
		}

		totalRecords = totRec
		if ipAddress != nil {
			item.IPAddress = *ipAddress
		}
		if len(detailsBytes) > 0 {
			_ = json.Unmarshal(detailsBytes, &item.Details)
		}
		if item.Details == nil {
			item.Details = make(map[string]any)
		}

		logs = append(logs, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit logs rows error: %w", err)
	}

	// Handle edge case where page > 1 yielded 0 rows but total count is non-zero
	if len(logs) == 0 && filter.Page > 1 {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
		var cnt int
		if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&cnt); err == nil {
			totalRecords = cnt
		}
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	return &domain.PaginatedAuditLogs{
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalRecords: totalRecords,
		Logs:         logs,
	}, nil
}
