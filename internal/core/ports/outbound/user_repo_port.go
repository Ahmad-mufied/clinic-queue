package outbound

import (
	"context"

	"clinic-queue/internal/core/domain"
)

// UserRepositoryPort defines the driven/outbound SPI interface for user data storage.
type UserRepositoryPort interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByID(ctx context.Context, id int) (*domain.User, error)
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
}
