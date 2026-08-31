package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/core/domain"

	"github.com/labstack/echo/v4"
)

func TestClientMetadataMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupHeaders   func(req *http.Request, rec *httptest.ResponseRecorder)
		remoteAddr     string
		expectedIP     string
		expectedUA     string
		expectedReqID  string
	}{
		{
			name: "Extract metadata with RequestID in response header",
			setupHeaders: func(req *http.Request, rec *httptest.ResponseRecorder) {
				req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser")
				req.Header.Set("X-Forwarded-For", "203.0.113.195")
				rec.Header().Set(echo.HeaderXRequestID, "resp-req-uuid-123")
			},
			remoteAddr:    "127.0.0.1:54321",
			expectedIP:    "203.0.113.195",
			expectedUA:    "Mozilla/5.0 TestBrowser",
			expectedReqID: "resp-req-uuid-123",
		},
		{
			name: "Extract metadata with RequestID in request header fallback",
			setupHeaders: func(req *http.Request, rec *httptest.ResponseRecorder) {
				req.Header.Set("User-Agent", "curl/7.68.0")
				req.Header.Set(echo.HeaderXRequestID, "req-header-uuid-456")
			},
			remoteAddr:    "192.168.1.50:12345",
			expectedIP:    "192.168.1.50",
			expectedUA:    "curl/7.68.0",
			expectedReqID: "req-header-uuid-456",
		},
		{
			name: "Extract metadata without custom headers",
			setupHeaders: func(req *http.Request, rec *httptest.ResponseRecorder) {
			},
			remoteAddr:    "10.0.0.2:8080",
			expectedIP:    "10.0.0.2",
			expectedUA:    "",
			expectedReqID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}
			rec := httptest.NewRecorder()

			if tt.setupHeaders != nil {
				tt.setupHeaders(req, rec)
			}

			c := e.NewContext(req, rec)

			var capturedMeta domain.ClientMetadata
			var capturedOK bool

			handler := func(ctx echo.Context) error {
				capturedMeta, capturedOK = domain.GetClientMetadata(ctx.Request().Context())
				return ctx.NoContent(http.StatusOK)
			}

			mw := middleware.ClientMetadataMiddleware()
			err := mw(handler)(c)
			if err != nil {
				t.Fatalf("unexpected middleware error: %v", err)
			}

			if !capturedOK {
				t.Fatal("expected client metadata in context, but was not found")
			}

			if capturedMeta.ClientIP != tt.expectedIP {
				t.Errorf("expected ClientIP %q, got %q", tt.expectedIP, capturedMeta.ClientIP)
			}
			if capturedMeta.UserAgent != tt.expectedUA {
				t.Errorf("expected UserAgent %q, got %q", tt.expectedUA, capturedMeta.UserAgent)
			}
			if capturedMeta.RequestID != tt.expectedReqID {
				t.Errorf("expected RequestID %q, got %q", tt.expectedReqID, capturedMeta.RequestID)
			}
		})
	}
}
