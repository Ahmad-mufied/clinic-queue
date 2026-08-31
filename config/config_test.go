package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default configuration
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_EXPIRATION_HOURS")
	os.Unsetenv("NATS_URL")
	os.Unsetenv("CASBIN_MODEL_PATH")
	os.Unsetenv("CASBIN_POLICY_PATH")
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}
	if cfg.JWTExpirationHours != 24 {
		t.Errorf("expected default JWTExpirationHours 24, got %d", cfg.JWTExpirationHours)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "http://localhost:3000" || cfg.CORSAllowedOrigins[1] != "http://localhost:8080" {
		t.Errorf("expected default CORSAllowedOrigins, got %+v", cfg.CORSAllowedOrigins)
	}

	// Test custom environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://custom:pass@localhost:5432/custom_db")
	os.Setenv("JWT_SECRET", "custom-secret")
	os.Setenv("JWT_EXPIRATION_HOURS", "48")
	os.Setenv("NATS_URL", "nats://localhost:4223")
	os.Setenv("CASBIN_MODEL_PATH", "custom_model.conf")
	os.Setenv("CASBIN_POLICY_PATH", "custom_policy.csv")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://clinic.example.com, https://admin.example.com")

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() with custom env error = %v", err)
	}

	if cfg.Port != "9090" || cfg.JWTExpirationHours != 48 || cfg.JWTSecret != "custom-secret" {
		t.Errorf("unexpected config with custom env: %+v", cfg)
	}
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://clinic.example.com" || cfg.CORSAllowedOrigins[1] != "https://admin.example.com" {
		t.Errorf("expected custom CORSAllowedOrigins, got %+v", cfg.CORSAllowedOrigins)
	}

	// Test empty or whitespace only CORS_ALLOWED_ORIGINS fallback
	os.Setenv("CORS_ALLOWED_ORIGINS", " , ")
	cfg, _ = LoadConfig()
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected fallback for empty CORS origins, got %+v", cfg.CORSAllowedOrigins)
	}

	// Test invalid JWT_EXPIRATION_HOURS fallback
	os.Setenv("JWT_EXPIRATION_HOURS", "invalid_number")
	cfg, _ = LoadConfig()
	if cfg.JWTExpirationHours != 24 {
		t.Errorf("expected fallback to 24 on invalid hours, got %d", cfg.JWTExpirationHours)
	}

	os.Setenv("JWT_EXPIRATION_HOURS", "-5")
	cfg, _ = LoadConfig()
	if cfg.JWTExpirationHours != 24 {
		t.Errorf("expected fallback to 24 on negative hours, got %d", cfg.JWTExpirationHours)
	}

	// Test RATE_LIMIT_ENABLED env variable
	os.Setenv("RATE_LIMIT_ENABLED", "false")
	cfg, _ = LoadConfig()
	if cfg.RateLimitEnabled != false {
		t.Errorf("expected RateLimitEnabled=false, got %v", cfg.RateLimitEnabled)
	}

	os.Setenv("RATE_LIMIT_ENABLED", "true")
	cfg, _ = LoadConfig()
	if cfg.RateLimitEnabled != true {
		t.Errorf("expected RateLimitEnabled=true, got %v", cfg.RateLimitEnabled)
	}
}

