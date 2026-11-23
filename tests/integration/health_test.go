//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/transport"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HealthIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
}

func TestHealthIntegrationSuite(t *testing.T) {
	suite.Run(t, new(HealthIntegrationTestSuite))
}

func (s *HealthIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup PostgreSQL testcontainer
	db := setupPostgres(s.T())

	cfg := &config.Config{
		ServiceName:    "api-template-test",
		ServiceVersion: testServiceVersion,
		Environment:    testEnvironment,
	}

	appLogger := logger.New("error", true)

	// Create health service with real database
	healthService := service.NewHealthService(cfg, db, appLogger)
	healthHandler := transport.NewHealthHandler(healthService)

	s.router = gin.New()
	healthHandler.RegisterRoutes(s.router)
}

func (s *HealthIntegrationTestSuite) TestHealthEndpoint() {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var response domain.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), domain.StatusHealthy, response.Status)
	assert.Equal(s.T(), "api-template-test", response.Service.Name)
	assert.Equal(s.T(), "1.0.0-test", response.Service.Version)
	assert.Equal(s.T(), "test", response.Service.Environment)
	assert.True(s.T(), response.Uptime > 0)

	// Check database check is healthy
	dbCheck, exists := response.Checks["database"]
	assert.True(s.T(), exists, "database check should exist")
	assert.Equal(s.T(), domain.StatusHealthy, dbCheck.Status)
	assert.Contains(s.T(), dbCheck.Message, "database connection successful")
}

func (s *HealthIntegrationTestSuite) TestReadinessEndpoint() {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var response domain.ReadinessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), domain.StatusHealthy, response.Status)
	assert.Equal(s.T(), "api-template-test", response.Service.Name)
}

func (s *HealthIntegrationTestSuite) TestLivenessEndpoint() {
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), "alive", response["status"])
	assert.NotNil(s.T(), response["timestamp"])
}
