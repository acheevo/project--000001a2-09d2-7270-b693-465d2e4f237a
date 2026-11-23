package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	authservice "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/transport"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/http"
	infoservice "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/service"
	infotransport "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/transport"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/database"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	appLogger := logger.New(cfg.LogLevel, cfg.IsDevelopment())

	db, err := database.New(cfg.DatabaseDSN(), cfg.IsDevelopment())
	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure database connection pool
	if err := db.SetConnectionPool(
		cfg.DBMaxIdleConns,
		cfg.DBMaxOpenConns,
		cfg.DBConnMaxLifetimeDuration(),
	); err != nil {
		appLogger.Error("failed to configure database connection pool", "error", err)
		return
	}

	// Database migration is optional for microservices
	// Remove or customize based on your service needs

	jwtService := authservice.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationDuration())

	healthService := service.NewHealthService(cfg, db, appLogger)
	healthHandler := transport.NewHealthHandler(healthService)

	infoSvc := infoservice.NewInfoService(cfg, db, appLogger)
	infoHandler := infotransport.NewInfoHandler(infoSvc)

	server := http.NewServer(cfg, appLogger, healthHandler, infoHandler, jwtService)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			appLogger.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	appLogger.Info("server started successfully")

	<-quit
	appLogger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		appLogger.Error("server forced to shutdown", "error", err)
	} else {
		appLogger.Info("server exited gracefully")
	}
}
