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

	// Test custom environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://custom:pass@localhost:5432/custom_db")
	os.Setenv("JWT_SECRET", "custom-secret")
	os.Setenv("JWT_EXPIRATION_HOURS", "48")
	os.Setenv("NATS_URL", "nats://localhost:4223")
	os.Setenv("CASBIN_MODEL_PATH", "custom_model.conf")
	os.Setenv("CASBIN_POLICY_PATH", "custom_policy.csv")

	cfg, err = LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() with custom env error = %v", err)
	}

	if cfg.Port != "9090" || cfg.JWTExpirationHours != 48 || cfg.JWTSecret != "custom-secret" {
		t.Errorf("unexpected config with custom env: %+v", cfg)
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
}
