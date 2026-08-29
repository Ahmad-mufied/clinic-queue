package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"clinic-queue/internal/core/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const testJWTSecret = "test-jwt-secret-key-987654321"

func getProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	// middleware dir is internal/adapters/inbound/middleware (4 levels deep from root)
	return filepath.Join(filepath.Dir(b), "../../../..")
}

func createTestToken(secret string, claims *domain.JWTCustomClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func TestJWTAuth(t *testing.T) {
	docID := 1
	validDoctorClaims := &domain.JWTCustomClaims{
		UserID:   1,
		Username: "doctor_a",
		Role:     domain.RoleDoctor,
		DoctorID: &docID,
		Name:     "Dr. Sarah Adams",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	validPatientClaims := &domain.JWTCustomClaims{
		UserID:   2,
		Username: "patient_john",
		Role:     domain.RolePatient,
		Name:     "John Doe",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	expiredClaims := &domain.JWTCustomClaims{
		UserID:   3,
		Username: "patient_expired",
		Role:     domain.RolePatient,
		Name:     "Expired Patient",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	// Create a token signed with SigningMethodNone
	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, validPatientClaims)
	noneTokenStr, _ := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	tests := []struct {
		name         string
		authHeader   string
		expectedCode int
		checkContext func(t *testing.T, c echo.Context)
	}{
		{
			name:         "Missing Authorization Header",
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Auth Header without Bearer Prefix",
			authHeader:   "Basic 12345",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Auth Header with empty Bearer token",
			authHeader:   "Bearer   ",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Invalid Malformed Token",
			authHeader:   "Bearer invalid.token.payload",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Token Signed with Wrong Secret",
			authHeader:   "Bearer " + createTestToken("wrong-secret", validDoctorClaims),
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Token Signed with None Algorithm",
			authHeader:   "Bearer " + noneTokenStr,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Expired Token",
			authHeader:   "Bearer " + createTestToken(testJWTSecret, expiredClaims),
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Valid Doctor Token",
			authHeader:   "Bearer " + createTestToken(testJWTSecret, validDoctorClaims),
			expectedCode: http.StatusOK,
			checkContext: func(t *testing.T, c echo.Context) {
				claims, ok := GetUserClaims(c)
				if !ok || claims.Username != "doctor_a" {
					t.Errorf("expected claims for doctor_a, got %+v", claims)
				}
				id, ok := GetUserID(c)
				if !ok || id != 1 {
					t.Errorf("expected user_id 1, got %d", id)
				}
				role, ok := GetUserRole(c)
				if !ok || role != "doctor" {
					t.Errorf("expected role doctor, got %s", role)
				}
				dID, ok := GetDoctorID(c)
				if !ok || dID == nil || *dID != 1 {
					t.Errorf("expected doctor_id 1, got %v", dID)
				}
			},
		},
		{
			name:         "Valid Patient Token",
			authHeader:   "Bearer " + createTestToken(testJWTSecret, validPatientClaims),
			expectedCode: http.StatusOK,
			checkContext: func(t *testing.T, c echo.Context) {
				claims, ok := GetUserClaims(c)
				if !ok || claims.Username != "patient_john" {
					t.Errorf("expected claims for patient_john, got %+v", claims)
				}
				dID, _ := GetDoctorID(c)
				if dID != nil {
					t.Errorf("expected nil doctor_id for patient, got %v", dID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := JWTAuth(testJWTSecret)(func(c echo.Context) error {
				if tt.checkContext != nil {
					tt.checkContext(t, c)
				}
				return c.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestContextHelpers_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	if claims, ok := GetUserClaims(c); ok || claims != nil {
		t.Errorf("expected false for empty claims, got %+v", claims)
	}
	if id, ok := GetUserID(c); ok || id != 0 {
		t.Errorf("expected false for empty user_id, got %d", id)
	}
	if role, ok := GetUserRole(c); ok || role != "" {
		t.Errorf("expected false for empty role, got %s", role)
	}
	if docID, ok := GetDoctorID(c); ok || docID != nil {
		t.Errorf("expected false for empty doctor_id, got %v", docID)
	}

	// Test with invalid type sets
	c.Set(ContextKeyUser, "invalid_claims_type")
	c.Set(ContextKeyUserID, "invalid_id_type")
	c.Set(ContextKeyRole, 12345)
	c.Set(ContextKeyDoctorID, "invalid_doctor_id_type")

	if _, ok := GetUserClaims(c); ok {
		t.Errorf("expected false for invalid claims type")
	}
	if _, ok := GetUserID(c); ok {
		t.Errorf("expected false for invalid user_id type")
	}
	if _, ok := GetUserRole(c); ok {
		t.Errorf("expected false for invalid role type")
	}
	if _, ok := GetDoctorID(c); ok {
		t.Errorf("expected false for invalid doctor_id type")
	}
}

func TestCasbinRBAC(t *testing.T) {
	rootDir := getProjectRoot()
	modelPath := filepath.Join(rootDir, "config/rbac_model.conf")
	policyPath := filepath.Join(rootDir, "config/rbac_policy.csv")

	enforcer, err := NewCasbinEnforcer(modelPath, policyPath)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	tests := []struct {
		name         string
		role         string
		path         string
		method       string
		expectedCode int
	}{
		{
			name:         "Public Access Login OK",
			role:         "",
			path:         "/api/auth/login",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Public Access Events OK",
			role:         "",
			path:         "/api/events",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Public Access Queue Status OK",
			role:         "",
			path:         "/api/queue/status",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Public Access Admin Stats Denied",
			role:         "",
			path:         "/api/admin/stats",
			method:       http.MethodGet,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Patient Access Queue Join OK",
			role:         "patient",
			path:         "/api/queue/join",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Patient Access My Ticket OK",
			role:         "patient",
			path:         "/api/queue/my-ticket",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Patient Access Admin Stats Denied",
			role:         "patient",
			path:         "/api/admin/stats",
			method:       http.MethodGet,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Patient Access Doctor Call-Next Denied",
			role:         "patient",
			path:         "/api/doctors/call-next",
			method:       http.MethodPost,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Doctor Access Status OK",
			role:         "doctor",
			path:         "/api/doctors/status",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Doctor Access Call-Next OK",
			role:         "doctor",
			path:         "/api/doctors/call-next",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Doctor Access Finish OK",
			role:         "doctor",
			path:         "/api/doctors/finish",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Doctor Access Admin Stats Denied",
			role:         "doctor",
			path:         "/api/admin/stats",
			method:       http.MethodGet,
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Admin Access Admin Stats OK",
			role:         "admin",
			path:         "/api/admin/stats",
			method:       http.MethodGet,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Admin Access Doctors OK",
			role:         "admin",
			path:         "/api/doctors/status",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Admin Access Queue OK",
			role:         "admin",
			path:         "/api/queue/reset",
			method:       http.MethodPost,
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.role != "" {
				c.Set(ContextKeyRole, tt.role)
			}

			handler := CasbinRBAC(enforcer)(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNewCasbinEnforcer_Error(t *testing.T) {
	_, err := NewCasbinEnforcer("non_existent_model.conf", "non_existent_policy.csv")
	if err == nil {
		t.Errorf("expected error for non-existent model and policy, got nil")
	}
}
