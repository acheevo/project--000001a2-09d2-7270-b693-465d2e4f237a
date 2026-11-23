package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	authservice "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/transport"
	infotransport "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/transport"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/managedai"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/middleware"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(cfg *config.Config, logger *slog.Logger, healthHandler *transport.HealthHandler,
	infoHandler *infotransport.InfoHandler, jwtService authservice.JWTService,
) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.Logging(logger))

	if cfg.MetricsEnabled {
		router.Use(middleware.Metrics())
		// Prometheus metrics endpoint
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Register health endpoints
	healthHandler.RegisterRoutes(router)

	jwtMiddleware := middleware.JWTAuth(jwtService, logger)

	// API routes for microservice endpoints
	api := router.Group(infotransport.BasePath)
	api.Use(jwtMiddleware) // Protect all API routes
	if ai := managedai.FromEnvironment(); ai != nil && managedai.SecureAppSigningKey(cfg.JWTSecret) {
		api.POST("/ai/run", gin.WrapH(ai))
	}
	contract := infoHandler.RegisterRoutes(router, api)
	if cfg.IsDevelopment() {
		router.GET("/openapi.json", func(c *gin.Context) {
			c.JSON(http.StatusOK, contract)
		})
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: min(5*time.Second, cfg.ReadTimeoutDuration()),
		ReadTimeout:       cfg.ReadTimeoutDuration(),
		WriteTimeout:      cfg.WriteTimeoutDuration(),
		IdleTimeout:       cfg.IdleTimeoutDuration(),
	}

	return &Server{
		server: server,
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "addr", s.server.Addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping HTTP server")
	return s.server.Shutdown(ctx)
}
