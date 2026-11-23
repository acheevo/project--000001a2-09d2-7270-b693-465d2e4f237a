//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CachingTestSuite struct {
	suite.Suite
	healthService service.HealthService
}

func TestCachingTestSuite(t *testing.T) {
	suite.Run(t, new(CachingTestSuite))
}

func (s *CachingTestSuite) SetupSuite() {
	// Setup PostgreSQL testcontainer
	db := setupPostgres(s.T())

	cfg := &config.Config{
		ServiceName:    "caching-test",
		ServiceVersion: testServiceVersion,
		Environment:    testEnvironment,
	}

	appLogger := logger.New("error", true)
	s.healthService = service.NewHealthService(cfg, db, appLogger)
}

func (s *CachingTestSuite) TestHealthCheckCaching() {
	// First health check - should hit database
	health1 := s.healthService.GetHealth()
	assert.Equal(s.T(), "healthy", string(health1.Status))

	dbCheck1, exists := health1.Checks["database"]
	assert.True(s.T(), exists)
	assert.Equal(s.T(), "healthy", string(dbCheck1.Status))
	firstTimestamp := dbCheck1.Timestamp

	// Second health check immediately - should use cache
	health2 := s.healthService.GetHealth()
	dbCheck2 := health2.Checks["database"]
	assert.Equal(s.T(), firstTimestamp, dbCheck2.Timestamp, "Timestamp should be the same due to caching")

	// Wait for cache to expire (30 seconds is too long for test, but we can verify logic)
	// For production, you'd wait: time.Sleep(31 * time.Second)
	// But for test, we just verify the caching mechanism works for immediate calls
}

func (s *CachingTestSuite) TestReadinessAlwaysFresh() {
	// First readiness check
	readiness1 := s.healthService.GetReadiness()
	assert.Equal(s.T(), "healthy", string(readiness1.Status))

	dbCheck1, exists := readiness1.Checks["database"]
	assert.True(s.T(), exists)
	firstTimestamp := dbCheck1.Timestamp

	// Small delay to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Second readiness check - should always be fresh
	readiness2 := s.healthService.GetReadiness()
	dbCheck2 := readiness2.Checks["database"]
	assert.True(s.T(), dbCheck2.Timestamp.After(firstTimestamp), "Readiness should always perform fresh checks")
}

func (s *CachingTestSuite) TestMixedHealthAndReadiness() {
	// Health check first (cached)
	health := s.healthService.GetHealth()
	healthDbCheck := health.Checks["database"]
	healthTimestamp := healthDbCheck.Timestamp

	// Readiness check (fresh)
	readiness := s.healthService.GetReadiness()
	readinessDbCheck := readiness.Checks["database"]

	// Readiness timestamp should be newer than health (fresh check)
	assert.True(s.T(), readinessDbCheck.Timestamp.After(healthTimestamp) ||
		readinessDbCheck.Timestamp.Equal(healthTimestamp))

	// Another health check immediately (should use cache)
	health2 := s.healthService.GetHealth()
	healthDbCheck2 := health2.Checks["database"]
	assert.Equal(s.T(), healthTimestamp, healthDbCheck2.Timestamp, "Health check should still use cached result")
}
