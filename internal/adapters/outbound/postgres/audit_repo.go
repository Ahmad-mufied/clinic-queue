package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/jackc/pgx/v5/pgconn"
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && log.UserID != nil {
			log.Details["unlinked_user_id"] = *log.UserID
			log.UserID = nil
			retryJSON, _ := json.Marshal(log.Details)
			retryErr := r.pool.QueryRow(
				ctx,
				query,
				nil,
				log.ActorName,
				log.Role,
				log.Action,
				retryJSON,
				log.IPAddress,
			).Scan(&log.ID, &log.CreatedAt)
			if retryErr == nil {
				return log, nil
			}
		}
		return nil, fmt.Errorf("insert audit log record: %w", err)
	}

	return log, nil
}

// QueryLogs retrieves paginated audit logs based on the provided filter parameters (supports Search, Dates, Roles, Actions, UserID, Cursor & Bidirectional Sorting).
func (r *AuditRepo) QueryLogs(ctx context.Context, filter domain.AuditLogFilter) (*domain.PaginatedAuditLogs, error) {
	filter.NormalizeSort()

	var conditions []string
	var args []any
	argIdx := 1

	if search := strings.TrimSpace(filter.Search); search != "" {
		searchPattern := "%" + search + "%"
		conditions = append(conditions, fmt.Sprintf("(actor_name ILIKE $%d OR ip_address ILIKE $%d OR action ILIKE $%d)", argIdx, argIdx+1, argIdx+2))
		args = append(args, searchPattern, searchPattern, searchPattern)
		argIdx += 3
	}

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

	if filter.UserID != nil && strings.TrimSpace(*filter.UserID) != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, strings.TrimSpace(*filter.UserID))
		argIdx++
	}

	if filter.StartDate != nil && !filter.StartDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil && !filter.EndDate.IsZero() {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	// Filter conditions for total count
	baseWhereClause := ""
	if len(conditions) > 0 {
		baseWhereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total records matching base filter
	var totalRecords int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", baseWhereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalRecords); err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	isAsc := filter.SortOrder == "asc"
	orderDir := "DESC"
	if isAsc {
		orderDir = "ASC"
	}

	// Cursor pagination clause
	if filter.Cursor != nil && strings.TrimSpace(*filter.Cursor) != "" {
		if isAsc {
			conditions = append(conditions, fmt.Sprintf("id > $%d", argIdx))
		} else {
			conditions = append(conditions, fmt.Sprintf("id < $%d", argIdx))
		}
		args = append(args, strings.TrimSpace(*filter.Cursor))
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Fetch limit + 1 to detect has_more and next_cursor
	fetchLimit := filter.Limit + 1
	var query string
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)

	if filter.Cursor != nil && strings.TrimSpace(*filter.Cursor) != "" {
		// Pure cursor query without offset
		query = fmt.Sprintf(`
			SELECT id, user_id, actor_name, role, action, details, ip_address, created_at
			FROM audit_logs
			%s
			ORDER BY id %s
			LIMIT $%d
		`, whereClause, orderDir, argIdx)
		queryArgs = append(queryArgs, fetchLimit)
	} else {
		// Offset fallback query
		query = fmt.Sprintf(`
			SELECT id, user_id, actor_name, role, action, details, ip_address, created_at
			FROM audit_logs
			%s
			ORDER BY id %s
			LIMIT $%d OFFSET $%d
		`, whereClause, orderDir, argIdx, argIdx+1)
		queryArgs = append(queryArgs, fetchLimit, filter.Offset())
	}

	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var (
			item         domain.AuditLog
			detailsBytes []byte
			ipAddress    *string
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
		); err != nil {
			return nil, fmt.Errorf("scan audit log row: %w", err)
		}

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

	hasMore := false
	var nextCursor *string
	if len(logs) > filter.Limit {
		hasMore = true
		logs = logs[:filter.Limit]
		lastItem := logs[len(logs)-1]
		nextCursor = &lastItem.ID
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	totalPages := 1
	if totalRecords > 0 && filter.Limit > 0 {
		totalPages = (totalRecords + filter.Limit - 1) / filter.Limit
	}

	return &domain.PaginatedAuditLogs{
		Page:         filter.Page,
		Limit:        filter.Limit,
		NextCursor:   nextCursor,
		HasMore:      hasMore,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		Logs:         logs,
	}, nil
}
