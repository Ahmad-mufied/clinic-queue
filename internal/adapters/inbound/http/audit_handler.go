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
}

// handleAuditError maps domain and validation errors to appropriate HTTP status responses.
func handleAuditError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidAction),
		errors.Is(err, domain.ErrInvalidPage),
		errors.Is(err, domain.ErrInvalidLimit):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input parameters"})
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

// GetAuditLogs handles GET /api/admin/audit-logs.
func (h *AuditHandler) GetAuditLogs(c echo.Context) error {
	page, err := parsePositiveQueryParam(c.QueryParam("page"), domain.DefaultPage)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid page parameter"})
	}

	limit, err := parsePositiveQueryParam(c.QueryParam("limit"), domain.DefaultLimit)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid limit parameter"})
	}

	var cursor *int
	if cursorStr := c.QueryParam("cursor"); cursorStr != "" {
		curVal, err := parsePositiveQueryParam(cursorStr, 0)
		if err != nil || curVal <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid cursor parameter"})
		}
		cursor = &curVal
	}

	var userID *int
	if uidStr := c.QueryParam("user_id"); uidStr != "" {
		uid, err := parsePositiveQueryParam(uidStr, 0)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user_id parameter"})
		}
		userID = &uid
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
	}

	result, err := h.auditUseCase.GetAuditLogs(c.Request().Context(), filter)
	if err != nil {
		return handleAuditError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}
