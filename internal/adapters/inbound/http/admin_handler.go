package http

import (
	"errors"
	"net/http"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

// AdminHandler handles HTTP requests for executive business analytics and doctor configuration management.
type AdminHandler struct {
	adminUseCase inbound.AdminUseCase
}

// NewAdminHandler constructs a new AdminHandler instance.
func NewAdminHandler(adminUseCase inbound.AdminUseCase) *AdminHandler {
	return &AdminHandler{adminUseCase: adminUseCase}
}

// RegisterRoutes registers all executive admin endpoints on the Echo router.
func (h *AdminHandler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc, rbacMW echo.MiddlewareFunc) {
	adminGroup := e.Group("/api/admin", authMW, rbacMW)
	adminGroup.GET("/stats", h.GetStats)
	adminGroup.POST("/doctors", h.UpdateDoctorConfig)
}

// handleAdminError maps domain and validation errors to appropriate HTTP responses.
func handleAdminError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input parameters"})
	case errors.Is(err, domain.ErrInvalidConsultationTime):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Average consultation time must be greater than 0"})
	case errors.Is(err, domain.ErrDoctorNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Doctor not found"})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
}

// GetStats handles GET /api/admin/stats.
func (h *AdminHandler) GetStats(c echo.Context) error {
	stats, err := h.adminUseCase.GetAnalyticsStats(c.Request().Context())
	if err != nil {
		return handleAdminError(c, err)
	}

	return c.JSON(http.StatusOK, stats)
}

// UpdateDoctorConfig handles POST /api/admin/doctors.
func (h *AdminHandler) UpdateDoctorConfig(c echo.Context) error {
	var req inbound.UpdateDoctorConfigDTO
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	doc, err := h.adminUseCase.UpdateDoctorConfig(c.Request().Context(), req)
	if err != nil {
		return handleAdminError(c, err)
	}

	return c.JSON(http.StatusOK, doc)
}
