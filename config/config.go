package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration variables for the application.
type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	NATSURL            string
	CasbinModelPath    string
	CasbinPolicyPath   string
	CORSAllowedOrigins []string
	RateLimitEnabled   bool
}

// LoadConfig loads configuration from environment variables and an optional .env file.
func LoadConfig() (*Config, error) {
	// Best-effort load .env file; ignore if missing
	_ = godotenv.Load()

	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgrespassword@localhost:5432/clinic_queue?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-clinic-jwt-key-change-in-prod")
	jwtExpHoursStr := getEnv("JWT_EXPIRATION_HOURS", "24")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	casbinModel := getEnv("CASBIN_MODEL_PATH", "config/rbac_model.conf")
	casbinPolicy := getEnv("CASBIN_POLICY_PATH", "config/rbac_policy.csv")
	corsOriginsStr := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")
	rateLimitEnabledStr := getEnv("RATE_LIMIT_ENABLED", "true")

	jwtExpHours, err := strconv.Atoi(jwtExpHoursStr)
	if err != nil || jwtExpHours <= 0 {
		jwtExpHours = 24
	}

	rateLimitEnabled := true
	if parsed, err := strconv.ParseBool(rateLimitEnabledStr); err == nil {
		rateLimitEnabled = parsed
	}

	var corsOrigins []string
	for _, origin := range strings.Split(corsOriginsStr, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			corsOrigins = append(corsOrigins, trimmed)
		}
	}
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: jwtExpHours,
		NATSURL:            natsURL,
		CasbinModelPath:    casbinModel,
		CasbinPolicyPath:   casbinPolicy,
		CORSAllowedOrigins: corsOrigins,
		RateLimitEnabled:   rateLimitEnabled,
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

