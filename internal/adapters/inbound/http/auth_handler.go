package http

import (
	"errors"
	"net/http"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles HTTP requests for user authentication and profiles.
type AuthHandler struct {
	authUseCase inbound.AuthUseCase
}

// NewAuthHandler constructs a new AuthHandler with the given inbound port.
func NewAuthHandler(authUseCase inbound.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

// RegisterRoutes registers the authentication routes on the provided Echo instance.
func (h *AuthHandler) RegisterRoutes(e *echo.Echo, authMW echo.MiddlewareFunc, rbacMW echo.MiddlewareFunc) {
	authGroup := e.Group("/api/auth")
	authGroup.POST("/login", h.Login)
	authGroup.POST("/register", h.Register)
	authGroup.GET("/me", h.GetMe, authMW, rbacMW)
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c echo.Context) error {
	var req inbound.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username and password are required",
		})
	}

	resp, err := h.authUseCase.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid username or password",
			})
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, resp)
}

// Register handles POST /api/auth/register.
func (h *AuthHandler) Register(c echo.Context) error {
	var req inbound.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	if req.Username == "" || req.Password == "" || req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username, password, and name are required",
		})
	}

	resp, err := h.authUseCase.Register(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrUsernameTaken) {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Username is already taken",
			})
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetMe handles GET /api/auth/me.
func (h *AuthHandler) GetMe(c echo.Context) error {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	user, err := h.authUseCase.GetProfile(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	return c.JSON(http.StatusOK, user)
}
