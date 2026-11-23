package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/database"
)

type HealthService interface {
	GetHealth() *domain.HealthResponse
	GetReadiness() *domain.ReadinessResponse
	AddCheck(check domain.Check)
	RemoveCheck(name string)
}

type healthService struct {
	cfg       *config.Config
	db        *database.Database
	logger    *slog.Logger
	startTime time.Time
	checks    map[string]domain.Check
	mu        sync.RWMutex
}

func NewHealthService(cfg *config.Config, db *database.Database, logger *slog.Logger) HealthService {
	service := &healthService{
		cfg:       cfg,
		db:        db,
		logger:    logger,
		startTime: time.Now(),
		checks:    make(map[string]domain.Check),
	}

	// Add default database check with caching
	if db != nil {
		service.AddCheck(NewCachedDatabaseCheck(db, 5*time.Minute))
	}

	return service
}

func (s *healthService) GetHealth() *domain.HealthResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := &domain.HealthResponse{
		Status:    domain.StatusHealthy,
		Service:   s.getServiceInfo(),
		Timestamp: time.Now(),
		Uptime:    time.Since(s.startTime),
		Checks:    make(map[string]domain.CheckInfo),
	}

	// Run all health checks (cached for performance)
	for name, check := range s.checks {
		var checkInfo domain.CheckInfo
		if cachedCheck, ok := check.(CachedCheck); ok {
			checkInfo = cachedCheck.CachedCheck()
		} else {
			checkInfo = check.Check()
		}
		health.Checks[name] = checkInfo

		if checkInfo.Status != domain.StatusHealthy {
			health.Status = domain.StatusDegraded
		}
	}

	return health
}

func (s *healthService) GetReadiness() *domain.ReadinessResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	readiness := &domain.ReadinessResponse{
		Status:    domain.StatusHealthy,
		Service:   s.getServiceInfo(),
		Timestamp: time.Now(),
		Checks:    make(map[string]domain.CheckInfo),
	}

	// Run all readiness checks (more strict than health, always fresh)
	for name, check := range s.checks {
		checkInfo := check.Check() // Always fresh check for readiness
		readiness.Checks[name] = checkInfo

		if checkInfo.Status != domain.StatusHealthy {
			readiness.Status = domain.StatusUnhealthy
			break
		}
	}

	return readiness
}

func (s *healthService) AddCheck(check domain.Check) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[check.Name()] = check
}

func (s *healthService) RemoveCheck(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.checks, name)
}

func (s *healthService) getServiceInfo() domain.ServiceInfo {
	return domain.ServiceInfo{
		Name:        s.cfg.ServiceName,
		Version:     s.cfg.ServiceVersion,
		Environment: s.cfg.Environment,
		Instance:    s.cfg.ServiceName + "-" + time.Now().Format("20060102150405"),
	}
}

// CachedCheck interface for checks that support caching
type CachedCheck interface {
	domain.Check
	CachedCheck() domain.CheckInfo
}

// CachedDatabaseCheck implements domain.Check with caching for database connectivity
type CachedDatabaseCheck struct {
	db         *database.Database
	lastCheck  time.Time
	lastResult domain.CheckInfo
	cacheTTL   time.Duration
	mu         sync.RWMutex
}

func NewCachedDatabaseCheck(db *database.Database, cacheTTL time.Duration) *CachedDatabaseCheck {
	return &CachedDatabaseCheck{
		db:       db,
		cacheTTL: cacheTTL,
	}
}

func (c *CachedDatabaseCheck) Name() string {
	return "database"
}

// Check performs a fresh database connectivity check
func (c *CachedDatabaseCheck) Check() domain.CheckInfo {
	return c.performCheck()
}

// CachedCheck returns cached result if still valid, otherwise performs fresh check
func (c *CachedDatabaseCheck) CachedCheck() domain.CheckInfo {
	c.mu.RLock()
	if time.Since(c.lastCheck) < c.cacheTTL && !c.lastCheck.IsZero() {
		result := c.lastResult
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	// Cache is expired or empty, perform fresh check
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check in case another goroutine updated the cache
	if time.Since(c.lastCheck) < c.cacheTTL && !c.lastCheck.IsZero() {
		return c.lastResult
	}

	// Perform fresh check and cache result
	c.lastResult = c.performCheck()
	c.lastCheck = time.Now()
	return c.lastResult
}

func (c *CachedDatabaseCheck) performCheck() domain.CheckInfo {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result int
	err := c.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	duration := time.Since(start)

	if err != nil {
		return domain.CheckInfo{
			Status:    domain.StatusUnhealthy,
			Message:   "database connection failed: " + err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return domain.CheckInfo{
		Status:    domain.StatusHealthy,
		Message:   "database connection successful",
		Duration:  duration,
		Timestamp: time.Now(),
	}
}
