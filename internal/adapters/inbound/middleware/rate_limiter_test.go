package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"clinic-queue/internal/adapters/inbound/middleware"

	"github.com/labstack/echo/v4"
)

func TestAuthRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := middleware.NewAuthRateLimiter(true)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	clientIP := "198.51.100.1"

	// First 5 requests within burst limit should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = clientIP + ":12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := limiter(handler)(c)
		if err != nil {
			t.Fatalf("request %d returned error: %v", i+1, err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d expected status 200, got %d", i+1, rec.Code)
		}
	}

	// 6th immediate request should exceed burst and return 429
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = clientIP + ":12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = limiter(handler)(c)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 on rate limit exceed, got %d", rec.Code)
	}

	// Another client IP should still be allowed
	otherIP := "198.51.100.2"
	reqOther := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	reqOther.RemoteAddr = otherIP + ":12345"
	recOther := httptest.NewRecorder()
	cOther := e.NewContext(reqOther, recOther)

	_ = limiter(handler)(cOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("expected status 200 for different IP, got %d", recOther.Code)
	}

	// Disabled limiter should allow all requests
	disabledLimiter := middleware.NewAuthRateLimiter(false)
	reqDisabled := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recDisabled := httptest.NewRecorder()
	cDisabled := e.NewContext(reqDisabled, recDisabled)
	if err := disabledLimiter(handler)(cDisabled); err != nil || recDisabled.Code != http.StatusOK {
		t.Fatalf("disabled rate limiter failed: %v", err)
	}
}

func TestQueueRateLimiter(t *testing.T) {
	e := echo.New()
	limiter := middleware.NewQueueRateLimiter(true)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	clientIP := "198.51.100.10"

	// First 10 requests within burst limit should succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/queue/join", nil)
		req.RemoteAddr = clientIP + ":12345"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := limiter(handler)(c)
		if err != nil {
			t.Fatalf("request %d returned error: %v", i+1, err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d expected status 200, got %d", i+1, rec.Code)
		}
	}

	// 11th immediate request should exceed burst and return 429
	req := httptest.NewRequest(http.MethodPost, "/api/queue/join", nil)
	req.RemoteAddr = clientIP + ":12345"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = limiter(handler)(c)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 on rate limit exceed, got %d", rec.Code)
	}

	// Another client IP should still be allowed
	otherIP := "198.51.100.20"
	reqOther := httptest.NewRequest(http.MethodPost, "/api/queue/join", nil)
	reqOther.RemoteAddr = otherIP + ":12345"
	recOther := httptest.NewRecorder()
	cOther := e.NewContext(reqOther, recOther)

	_ = limiter(handler)(cOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("expected status 200 for different IP, got %d", recOther.Code)
	}

	// Disabled limiter should allow all requests
	disabledLimiter := middleware.NewQueueRateLimiter(false)
	reqDisabled := httptest.NewRequest(http.MethodPost, "/api/queue/join", nil)
	recDisabled := httptest.NewRecorder()
	cDisabled := e.NewContext(reqDisabled, recDisabled)
	if err := disabledLimiter(handler)(cDisabled); err != nil || recDisabled.Code != http.StatusOK {
		t.Fatalf("disabled rate limiter failed: %v", err)
	}
}
