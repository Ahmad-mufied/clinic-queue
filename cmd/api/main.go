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
	natsAdapter "clinic-queue/internal/adapters/outbound/nats"
	"clinic-queue/internal/adapters/outbound/postgres"
	"clinic-queue/config"
	"clinic-queue/internal/core/ports/outbound"
	"clinic-queue/internal/core/usecase"

	"github.com/labstack/echo/v4"
	echoMW "github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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

	// 4. Initialize NATS JetStream Client
	var nc *nats.Conn
	var js jetstream.JetStream
	nc, js, err = natsAdapter.NewNATSClient(cfg.NATSURL)
	if err != nil {
		log.Printf("Warning: NATS JetStream connection failed (operating in fallback mode): %v", err)
	} else {
		defer nc.Close()
		log.Println("Successfully connected to NATS JetStream")
	}

	// 5. Initialize Casbin RBAC Enforcer
	enforcer, err := customMW.NewCasbinEnforcer(cfg.CasbinModelPath, cfg.CasbinPolicyPath)
	if err != nil {
		log.Fatalf("Failed to initialize Casbin RBAC enforcer: %v", err)
	}
	log.Println("Casbin RBAC enforcer initialized successfully")

	// 6. Dependency Injection / Wiring (Hexagonal Ports & Adapters)
	userRepo := postgres.NewUserRepo(dbPool)
	queueRepo := postgres.NewQueueRepo(dbPool)
	doctorRepo := postgres.NewDoctorRepo(dbPool)
	consultationRepo := postgres.NewConsultationRepo(dbPool)
	analyticsRepo := postgres.NewAnalyticsRepo(dbPool)
	auditRepo := postgres.NewAuditRepo(dbPool)

	var eventPublisher outbound.EventPublisherPort = natsAdapter.NewNATSEventPublisher(nc, js)

	jwtExpiration := time.Duration(cfg.JWTExpirationHours) * time.Hour
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.JWTSecret, jwtExpiration)
	queueUseCase := usecase.NewQueueUseCase(queueRepo, doctorRepo, eventPublisher)
	doctorUseCase := usecase.NewDoctorUseCase(doctorRepo, consultationRepo, eventPublisher)
	adminUseCase := usecase.NewAdminUseCase(analyticsRepo, doctorRepo, eventPublisher)
	auditUseCase := usecase.NewAuditUseCase(auditRepo, eventPublisher)

	authHandler := httpAdapter.NewAuthHandler(authUseCase)
	queueHandler := httpAdapter.NewQueueHandler(queueUseCase)
	doctorHandler := httpAdapter.NewDoctorHandler(doctorUseCase)
	adminHandler := httpAdapter.NewAdminHandler(adminUseCase)
	auditHandler := httpAdapter.NewAuditHandler(auditUseCase)
	sseHandler := httpAdapter.NewSSEHandler()

	if nc != nil {
		if _, err := sseHandler.ListenToNATS(context.Background(), nc, "clinic.>"); err != nil {
			log.Printf("Warning: failed to subscribe SSE handler to NATS: %v", err)
		} else {
			log.Println("SSE Broadcaster subscribed to NATS stream clinic.>")
		}
	}

	// 7. Initialize Echo Router & Middlewares
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

	// 8. Register Route Handlers
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "smart-clinic-queue-api",
		})
	})

	authHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)
	queueHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)
	doctorHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)
	adminHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)
	auditHandler.RegisterRoutes(e, jwtAuthMW, casbinRBACMW)
	sseHandler.RegisterRoutes(e, casbinRBACMW)


	// 9. Start HTTP Server with Graceful Shutdown
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
