package inbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// LoginRequest defines the parameters required to authenticate a user.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest defines the parameters required to register a new patient.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// AuthResponse defines the payload returned upon successful authentication or registration.
type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// AuthUseCase defines the driving/inbound port for authentication and user profile operations.
type AuthUseCase interface {
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
}
