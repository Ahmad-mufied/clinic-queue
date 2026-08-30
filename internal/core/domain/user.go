package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role represents the role of a user in the clinic system.
type Role string

const (
	RolePatient Role = "patient"
	RoleDoctor  Role = "doctor"
	RoleAdmin   Role = "admin"
)

// IsValid checks whether the role is one of the valid enum values.
func (r Role) IsValid() bool {
	switch r {
	case RolePatient, RoleDoctor, RoleAdmin:
		return true
	default:
		return false
	}
}

// User represents a user entity in the clinic queue system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Role         Role      `json:"role"`
	DoctorID     *string   `json:"doctor_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JWTCustomClaims represents the payload claims stored in a signed JWT token.
type JWTCustomClaims struct {
	UserID   string  `json:"user_id"`
	Username string  `json:"username"`
	Role     Role    `json:"role"`
	DoctorID *string `json:"doctor_id,omitempty"`
	Name     string  `json:"name"`
	jwt.RegisteredClaims
}
