package http

import (
	"errors"
	"net/http"
	"strings"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

// QueueHandler handles HTTP requests for patient queue operations and status polling.
type QueueHandler struct {
	queueUseCase inbound.QueueUseCase
}

// NewQueueHandler constructs a new QueueHandler.
func NewQueueHandler(queueUseCase inbound.QueueUseCase) *QueueHandler {
	return &QueueHandler{queueUseCase: queueUseCase}
}

// RegisterRoutes registers the queue routes on the Echo instance.
func (h *QueueHandler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc, rbacMW echo.MiddlewareFunc) {
	queueGroup := e.Group("/api/queue")
	queueGroup.POST("/join", h.JoinQueue, authMW, rbacMW)
	queueGroup.GET("/my-ticket", h.GetMyTicket, authMW, rbacMW)
	queueGroup.GET("/status", h.GetQueueStatus, rbacMW)
}

// handleQueueError maps domain errors to appropriate HTTP responses.
func handleQueueError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Patient name is required"})
	case errors.Is(err, domain.ErrActiveTicketExists):
		return c.JSON(http.StatusConflict, map[string]string{"error": "Active queue ticket already exists"})
	case errors.Is(err, domain.ErrNoDoctorsAvailable):
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "No doctors currently configured for this clinic"})
	case errors.Is(err, domain.ErrTicketNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No active ticket found"})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
}

// JoinQueue handles POST /api/queue/join.
func (h *QueueHandler) JoinQueue(c echo.Context) error {
	var req inbound.JoinQueueRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	trimmedName := strings.TrimSpace(req.PatientName)
	if trimmedName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Patient name is required",
		})
	}

	var userID *string
	if id, ok := middleware.GetUserID(c); ok && strings.TrimSpace(id) != "" {
		trimmedID := strings.TrimSpace(id)
		userID = &trimmedID
	}

	ticket, err := h.queueUseCase.JoinQueue(c.Request().Context(), userID, trimmedName)
	if err != nil {
		return handleQueueError(c, err)
	}

	return c.JSON(http.StatusCreated, inbound.JoinQueueResponse{
		Ticket: ticket,
	})
}

// GetMyTicket handles GET /api/queue/my-ticket.
func (h *QueueHandler) GetMyTicket(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok || strings.TrimSpace(userID) == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	ticket, err := h.queueUseCase.GetMyTicket(c.Request().Context(), userID)
	if err != nil {
		return handleQueueError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"ticket": ticket,
	})
}

// GetQueueStatus handles GET /api/queue/status.
func (h *QueueHandler) GetQueueStatus(c echo.Context) error {
	status, err := h.queueUseCase.GetQueueStatus(c.Request().Context())
	if err != nil {
		return handleQueueError(c, err)
	}

	return c.JSON(http.StatusOK, status)
}
