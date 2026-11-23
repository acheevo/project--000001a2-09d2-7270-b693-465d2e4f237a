//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authservice "github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/service"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/info/transport"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/middleware"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/config"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type InfoIntegrationTestSuite struct {
	suite.Suite
	router     *gin.Engine
	jwtService authservice.JWTService
}

func TestInfoIntegrationSuite(t *testing.T) {
	suite.Run(t, new(InfoIntegrationTestSuite))
}

func (s *InfoIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup PostgreSQL testcontainer
	db := setupPostgres(s.T())

	cfg := &config.Config{
		ServiceName:    "api-template-test",
		ServiceVersion: testServiceVersion,
		Environment:    testEnvironment,
		JWTSecret:      "test-secret",
	}

	appLogger := logger.New("error", true)
	s.jwtService = authservice.NewJWTService(cfg.JWTSecret, 24*time.Hour)

	// Create info service with real database
	infoService := service.NewInfoService(cfg, db, appLogger)
	infoHandler := transport.NewInfoHandler(infoService)

	s.router = gin.New()

	// Set up JWT middleware
	jwtMiddleware := middleware.JWTAuth(s.jwtService, appLogger)

	api := s.router.Group("/api/v1")
	api.Use(jwtMiddleware)
	infoHandler.RegisterRoutes(s.router, api)
}

func (s *InfoIntegrationTestSuite) TestInfoEndpoint() {
	// Generate a valid JWT token
	token, err := s.jwtService.GenerateServiceToken("test-service", "test-id", "1.0.0", "test")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var response domain.InfoResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(s.T(), err)

	// Check service info
	assert.Equal(s.T(), "api-template-test", response.Service.Name)
	assert.Equal(s.T(), "1.0.0-test", response.Service.Version)
	assert.Equal(s.T(), "test", response.Service.Environment)
	assert.NotEmpty(s.T(), response.Service.Instance)
	assert.NotEmpty(s.T(), response.Service.Uptime)

	// Check system info
	assert.NotEmpty(s.T(), response.System.GoVersion)
	assert.NotEmpty(s.T(), response.System.Platform)
	assert.Greater(s.T(), response.System.NumCPU, 0)
	assert.Greater(s.T(), response.System.NumGoroutine, 0)

	// Check dependencies (database should be healthy)
	assert.Equal(s.T(), "healthy", response.Dependencies.Database.Status)
	assert.True(s.T(), response.Dependencies.Database.Reachable)
	assert.Empty(s.T(), response.Dependencies.Database.Error)

	// Check timestamp
	assert.True(s.T(), response.Timestamp.Before(time.Now().Add(time.Second)))
}

func (s *InfoIntegrationTestSuite) TestInfoEndpointUnauthorized() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

func (s *InfoIntegrationTestSuite) TestInfoEndpointInvalidToken() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}
