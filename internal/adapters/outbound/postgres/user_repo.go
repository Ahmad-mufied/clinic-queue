package postgres

import (
	"context"
	"errors"
	"fmt"

	"clinic-queue/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo implements the outbound.UserRepositoryPort interface using PostgreSQL 18 via pgx/v5.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo constructs a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// FindByUsername queries a single user by their unique username.
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash, name, role, doctor_id, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var user domain.User
	var roleStr string
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&roleStr,
		&user.DoctorID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by username %s: %w", username, err)
	}

	user.Role = domain.Role(roleStr)
	return &user, nil
}

// FindByID queries a single user by their primary key ID.
func (r *UserRepo) FindByID(ctx context.Context, id int) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash, name, role, doctor_id, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var roleStr string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&roleStr,
		&user.DoctorID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id %d: %w", id, err)
	}

	user.Role = domain.Role(roleStr)
	return &user, nil
}

// CreateUser inserts a new user record and returns the persisted entity with its generated ID and timestamps.
func (r *UserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (username, password_hash, name, role, doctor_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Name,
		string(user.Role),
		user.DoctorID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert user %s: %w", user.Username, err)
	}

	return user, nil
}
