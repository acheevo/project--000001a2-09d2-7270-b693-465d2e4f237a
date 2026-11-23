package service

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/database"
)

type InfoService interface {
	GetInfo(context.Context) *domain.InfoResponse
}

type infoService struct {
	cfg       *config.Config
	db        *database.Database
	logger    *slog.Logger
	startTime time.Time
}

func NewInfoService(cfg *config.Config, db *database.Database, logger *slog.Logger) InfoService {
	return &infoService{
		cfg:       cfg,
		db:        db,
		logger:    logger,
		startTime: time.Now(),
	}
}

func (s *infoService) GetInfo(ctx context.Context) *domain.InfoResponse {
	now := time.Now()

	return &domain.InfoResponse{
		Service:      s.getServiceInfo(now),
		System:       s.getSystemInfo(),
		Dependencies: s.getDependencyStatus(ctx),
		Timestamp:    now,
	}
}

func (s *infoService) getServiceInfo(now time.Time) domain.ServiceInfo {
	uptime := now.Sub(s.startTime)

	return domain.ServiceInfo{
		Name:        s.cfg.ServiceName,
		Version:     s.cfg.ServiceVersion,
		Environment: s.cfg.Environment,
		Instance:    s.cfg.GetServiceID(),
		Uptime:      uptime.String(),
	}
}

func (s *infoService) getSystemInfo() domain.SystemInfo {
	return domain.SystemInfo{
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

func (s *infoService) getDependencyStatus(ctx context.Context) domain.DependencyStatus {
	return domain.DependencyStatus{
		Database: s.checkDatabaseStatus(ctx),
	}
}

func (s *infoService) checkDatabaseStatus(ctx context.Context) domain.DatabaseStatus {
	if s.db == nil {
		return domain.DatabaseStatus{
			Status:    "disabled",
			Reachable: false,
			Error:     "database not configured",
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result int
	err := s.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	responseTime := time.Since(start)

	if err != nil {
		s.logger.Warn("database health check failed", "error", err, "response_time", responseTime)
		return domain.DatabaseStatus{
			Status:       "unhealthy",
			Reachable:    false,
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}

	return domain.DatabaseStatus{
		Status:       "healthy",
		Reachable:    true,
		ResponseTime: responseTime,
	}
}
