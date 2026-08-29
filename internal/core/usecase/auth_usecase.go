package usecase

import (
	"context"
	"fmt"
	"time"

	"clinic-queue/internal/core/domain"
	"clinic-queue/internal/core/ports/inbound"
	"clinic-queue/internal/core/ports/outbound"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthUseCase implements the inbound.AuthUseCase port.
type AuthUseCase struct {
	userRepo      outbound.UserRepositoryPort
	jwtSecret     string
	jwtExpiration time.Duration
}

// NewAuthUseCase constructs a new AuthUseCase instance.
func NewAuthUseCase(userRepo outbound.UserRepositoryPort, jwtSecret string, jwtExpiration time.Duration) *AuthUseCase {
	return &AuthUseCase{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
	}
}

// Login authenticates a user and returns a signed JWT token and user entity.
func (u *AuthUseCase) Login(ctx context.Context, req inbound.LoginRequest) (*inbound.AuthResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := u.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	if user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token := u.generateToken(user)

	return &inbound.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

// Register registers a new patient user, saves them to repository, and returns a signed JWT.
func (u *AuthUseCase) Register(ctx context.Context, req inbound.RegisterRequest) (*inbound.AuthResponse, error) {
	if req.Username == "" || req.Password == "" || req.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	existing, err := u.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("check username availability: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrUsernameTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser := &domain.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         domain.RolePatient,
	}

	createdUser, err := u.userRepo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token := u.generateToken(createdUser)

	return &inbound.AuthResponse{
		Token: token,
		User:  createdUser,
	}, nil
}

// GetProfile retrieves a user by ID and returns the sanitized domain model.
func (u *AuthUseCase) GetProfile(ctx context.Context, userID int) (*domain.User, error) {
	if userID <= 0 {
		return nil, domain.ErrInvalidInput
	}

	user, err := u.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (u *AuthUseCase) generateToken(user *domain.User) string {
	claims := &domain.JWTCustomClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		DoctorID:  user.DoctorID,
		Name:      user.Name,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(u.jwtExpiration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(u.jwtSecret))
	return tokenStr
}
