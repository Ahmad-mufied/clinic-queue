package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "clinic-queue/internal/adapters/inbound/http"
	customMW "clinic-queue/internal/adapters/inbound/middleware"
	"clinic-queue/internal/adapters/outbound/postgres"
	"clinic-queue/config"
	"clinic-queue/internal/core/usecase"

	"github.com/labstack/echo/v4"
	echoMW "github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Load Application Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. Initialize PostgreSQL Connection Pool
	dbPool, err := postgres.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Database connection failed (will retry or start in degraded mode if needed): %v", err)
	} else {
		defer dbPool.Close()
		log.Println("Successfully connected to PostgreSQL 18 database")

		// 3. Run Goose Database Auto-Migrations
		if err := postgres.RunDatabaseMigrations(dbPool); err != nil {
			log.Fatalf("Failed to execute database migrations: %v", err)
		}
		log.Println("Database schema & demo seed migrations executed successfully")
	}

	// 4. Initialize Casbin RBAC Enforcer
	enforcer, err := customMW.NewCasbinEnforcer(cfg.CasbinModelPath, cfg.CasbinPolicyPath)
	if err != nil {
		log.Fatalf("Failed to initialize Casbin RBAC enforcer: %v", err)
	}
	log.Println("Casbin RBAC enforcer initialized successfully")

	// 5. Dependency Injection / Wiring (Hexagonal Ports & Adapters)
	userRepo := postgres.NewUserRepo(dbPool)
	jwtExpiration := time.Duration(cfg.JWTExpirationHours) * time.Hour
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.JWTSecret, jwtExpiration)
	authHandler := httpAdapter.NewAuthHandler(authUseCase)

	// 6. Initialize Echo Router & Middlewares
	e := echo.New()
	e.HideBanner = true

	e.Use(echoMW.Logger())
	e.Use(echoMW.Recover())
	e.Use(echoMW.CORSWithConfig(echoMW.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	jwtAuthMW := customMW.JWTAuth(cfg.JWTSecret)
	casbinRBACMW := customMW.CasbinRBAC(enforcer)

	// 7. Register Route Handlers
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "smart-clinic-queue-api",
		})
	})

	authHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)

	// Protected routes for RBAC verification
	apiGroup := e.Group("/api", jwtAuthMW, casbinRBACMW)
	apiGroup.GET("/admin/stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Admin stats placeholder"})
	})
	apiGroup.POST("/doctors/call-next", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Doctor call-next placeholder"})
	})
	apiGroup.POST("/queue/join", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Patient queue join placeholder"})
	})

	// 8. Start HTTP Server with Graceful Shutdown
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Starting Smart Clinic Queue API server on port %s...", cfg.Port)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
