package http

import (
	"errors"
	"net/http"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

// DoctorHandler handles HTTP requests for doctor shift toggling, patient queue calling, and consultation completion.
type DoctorHandler struct {
	doctorUseCase inbound.DoctorUseCase
}

// NewDoctorHandler constructs a new DoctorHandler instance.
func NewDoctorHandler(doctorUseCase inbound.DoctorUseCase) *DoctorHandler {
	return &DoctorHandler{doctorUseCase: doctorUseCase}
}

// RegisterRoutes registers all doctor workspace endpoints on the Echo router.
func (h *DoctorHandler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc, rbacMW echo.MiddlewareFunc) {
	doctorGroup := e.Group("/api/doctors", authMW, rbacMW)
	doctorGroup.POST("/status", h.ToggleStatus)
	doctorGroup.POST("/call-next", h.CallNextPatient)
	doctorGroup.POST("/finish", h.FinishConsultation)
	doctorGroup.GET("/workspace", h.GetWorkspace)
}

// getContextDoctorID retrieves and validates the doctor ID from the request context claims.
func (h *DoctorHandler) getContextDoctorID(c echo.Context) (*int, error) {
	docID, ok := middleware.GetDoctorID(c)
	if !ok || docID == nil || *docID <= 0 {
		return nil, domain.ErrDoctorProfileMissing
	}
	return docID, nil
}

// handleDoctorError maps domain errors to appropriate HTTP responses.
func handleDoctorError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrDoctorProfileMissing):
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Doctor profile required"})
	case errors.Is(err, domain.ErrDoctorOffline):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Doctor must be online to call patients"})
	case errors.Is(err, domain.ErrActiveConsultationExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": "Active consultation already in progress. Finish current session first."})
	case errors.Is(err, domain.ErrNoActiveConsultation):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No active consultation found to finish"})
	case errors.Is(err, domain.ErrQueueEmpty):
		return c.JSON(http.StatusOK, map[string]string{"message": "Queue is empty. No patients waiting."})
	case errors.Is(err, domain.ErrDoctorNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Doctor not found"})
	case errors.Is(err, domain.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid doctor ID"})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
}

// ToggleStatus handles POST /api/doctors/status.
func (h *DoctorHandler) ToggleStatus(c echo.Context) error {
	docID, err := h.getContextDoctorID(c)
	if err != nil {
		return handleDoctorError(c, err)
	}

	var req inbound.ToggleShiftRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	resp, err := h.doctorUseCase.ToggleStatus(c.Request().Context(), *docID, req.IsOnline)
	if err != nil {
		return handleDoctorError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// CallNextPatient handles POST /api/doctors/call-next.
func (h *DoctorHandler) CallNextPatient(c echo.Context) error {
	docID, err := h.getContextDoctorID(c)
	if err != nil {
		return handleDoctorError(c, err)
	}

	session, err := h.doctorUseCase.CallNextPatient(c.Request().Context(), *docID)
	if err != nil {
		return handleDoctorError(c, err)
	}

	return c.JSON(http.StatusOK, session)
}

// FinishConsultation handles POST /api/doctors/finish.
func (h *DoctorHandler) FinishConsultation(c echo.Context) error {
	docID, err := h.getContextDoctorID(c)
	if err != nil {
		return handleDoctorError(c, err)
	}

	resp, err := h.doctorUseCase.FinishConsultation(c.Request().Context(), *docID)
	if err != nil {
		return handleDoctorError(c, err)
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkspace handles GET /api/doctors/workspace.
func (h *DoctorHandler) GetWorkspace(c echo.Context) error {
	docID, err := h.getContextDoctorID(c)
	if err != nil {
		return handleDoctorError(c, err)
	}

	workspace, err := h.doctorUseCase.GetWorkspace(c.Request().Context(), *docID)
	if err != nil {
		return handleDoctorError(c, err)
	}

	return c.JSON(http.StatusOK, workspace)
}
