package http

import (
	"errors"
	"net/http"
	"strconv"

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

	filter := domain.AuditLogFilter{
		Action: c.QueryParam("action"),
		Role:   c.QueryParam("role"),
		Cursor: cursor,
		Page:   page,
		Limit:  limit,
	}

	result, err := h.auditUseCase.GetAuditLogs(c.Request().Context(), filter)
	if err != nil {
		return handleAuditError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}
