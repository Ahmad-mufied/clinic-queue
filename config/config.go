package config

import (
	"os"
	"strconv"

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

	jwtExpHours, err := strconv.Atoi(jwtExpHoursStr)
	if err != nil || jwtExpHours <= 0 {
		jwtExpHours = 24
	}

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: jwtExpHours,
		NATSURL:            natsURL,
		CasbinModelPath:    casbinModel,
		CasbinPolicyPath:   casbinPolicy,
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
