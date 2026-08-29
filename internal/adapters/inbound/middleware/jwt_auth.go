package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"clinic-queue/internal/core/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const (
	ContextKeyUser     = "user"
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyRole     = "role"
	ContextKeyDoctorID = "doctor_id"
)

// JWTAuth returns an Echo middleware that validates the Bearer JWT token in the Authorization header.
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing or malformed JWT token",
				})
			}

			tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if tokenStr == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing or malformed JWT token",
				})
			}

			claims := &domain.JWTCustomClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired JWT token",
				})
			}

			c.Set(ContextKeyUser, claims)
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUsername, claims.Username)
			c.Set(ContextKeyRole, string(claims.Role))
			c.Set(ContextKeyDoctorID, claims.DoctorID)

			return next(c)
		}
	}
}

// GetUserClaims extracts the JWTCustomClaims stored in the Echo context.
func GetUserClaims(c echo.Context) (*domain.JWTCustomClaims, bool) {
	val := c.Get(ContextKeyUser)
	if val == nil {
		return nil, false
	}
	claims, ok := val.(*domain.JWTCustomClaims)
	return claims, ok
}

// GetUserID extracts the user ID stored in the Echo context.
func GetUserID(c echo.Context) (int, bool) {
	val := c.Get(ContextKeyUserID)
	if val == nil {
		return 0, false
	}
	id, ok := val.(int)
	return id, ok
}

// GetUserRole extracts the user role stored in the Echo context.
func GetUserRole(c echo.Context) (string, bool) {
	val := c.Get(ContextKeyRole)
	if val == nil {
		return "", false
	}
	role, ok := val.(string)
	return role, ok
}

// GetDoctorID extracts the optional doctor ID stored in the Echo context.
func GetDoctorID(c echo.Context) (*int, bool) {
	val := c.Get(ContextKeyDoctorID)
	if val == nil {
		return nil, false
	}
	docID, ok := val.(*int)
	return docID, ok
}
