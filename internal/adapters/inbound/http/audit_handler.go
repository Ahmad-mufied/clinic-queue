package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

// AuditHandler handles HTTP requests for audit trail reporting and activity logs inspection.
type AuditHandler struct {
	auditUseCase inbound.AuditUseCase
}

// NewAuditHandler constructs a new AuditHandler instance.
func NewAuditHandler(auditUseCase inbound.AuditUseCase) *AuditHandler {
	return &AuditHandler{auditUseCase: auditUseCase}
}

// RegisterRoutes registers audit trail admin endpoints on the Echo router.
func (h *AuditHandler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc, rbacMW echo.MiddlewareFunc) {
	adminGroup := e.Group("/api/admin", authMW, rbacMW)
	adminGroup.GET("/audit-logs", h.GetAuditLogs)
	adminGroup.GET("/audit-logs/:id", h.GetAuditLogByID)
}

// handleAuditError maps domain and validation errors to appropriate HTTP status responses.
func handleAuditError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidAction),
		errors.Is(err, domain.ErrInvalidPage),
		errors.Is(err, domain.ErrInvalidLimit):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input parameters"})
	case errors.Is(err, domain.ErrAuditLogNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Audit log not found"})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
}

// parsePositiveQueryParam parses an optional query parameter string into a positive integer.
func parsePositiveQueryParam(raw string, defaultVal int) (int, error) {
	if raw == "" {
		return defaultVal, nil
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return 0, domain.ErrInvalidInput
	}
	return val, nil
}

// parseOptionalTime parses a date string supporting RFC3339 and date-only YYYY-MM-DD formats.
func parseOptionalTime(raw string, isEnd bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, raw); err == nil {
			if format == "2006-01-02" && isEnd {
				endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
				return &endOfDay, nil
			}
			return &t, nil
		}
	}

	return nil, domain.ErrInvalidInput
}

// isValidUUID validates standard 36-character hexadecimal UUID format (8-4-4-4-12).
func isValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	for i, r := range u {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

// GetAuditLogs handles GET /api/admin/audit-logs.
func (h *AuditHandler) GetAuditLogs(c echo.Context) error {
	showAll := strings.EqualFold(c.QueryParam("all"), "true") || c.QueryParam("all") == "1"

	defaultLimit := domain.DefaultLimit
	if showAll {
		defaultLimit = domain.MaxFullExportLimit
	}

	page, err := parsePositiveQueryParam(c.QueryParam("page"), domain.DefaultPage)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid page parameter"})
	}

	limit, err := parsePositiveQueryParam(c.QueryParam("limit"), defaultLimit)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid limit parameter"})
	}

	var cursor *string
	if cursorStr := strings.TrimSpace(c.QueryParam("cursor")); cursorStr != "" {
		if !isValidUUID(cursorStr) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid cursor parameter"})
		}
		cursor = &cursorStr
	}

	var userID *string
	if uidStr := strings.TrimSpace(c.QueryParam("user_id")); uidStr != "" {
		userID = &uidStr
	}

	fromStr := c.QueryParam("from")
	if fromStr == "" {
		fromStr = c.QueryParam("start_date")
	}
	startDate, err := parseOptionalTime(fromStr, false)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid start_date parameter"})
	}

	toStr := c.QueryParam("to")
	if toStr == "" {
		toStr = c.QueryParam("end_date")
	}
	endDate, err := parseOptionalTime(toStr, true)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid end_date parameter"})
	}

	order := c.QueryParam("order")
	if order == "" {
		order = c.QueryParam("sort_order")
	}

	filter := domain.AuditLogFilter{
		Search:    c.QueryParam("search"),
		Action:    c.QueryParam("action"),
		Role:      c.QueryParam("role"),
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
		SortOrder: order,
		Cursor:    cursor,
		Page:      page,
		Limit:     limit,
		All:       showAll,
	}

	result, err := h.auditUseCase.GetAuditLogs(c.Request().Context(), filter)
	if err != nil {
		return handleAuditError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

// GetAuditLogByID handles GET /api/admin/audit-logs/:id for deep forensic context and JSON details.
func (h *AuditHandler) GetAuditLogByID(c echo.Context) error {
	id := strings.TrimSpace(c.Param("id"))
	if !isValidUUID(id) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid audit log ID format"})
	}

	log, err := h.auditUseCase.GetAuditLogByID(c.Request().Context(), id)
	if err != nil {
		return handleAuditError(c, err)
	}

	return c.JSON(http.StatusOK, log)
}

