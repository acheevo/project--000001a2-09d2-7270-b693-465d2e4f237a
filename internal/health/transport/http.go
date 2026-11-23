package transport

import (
	"net/http"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/domain"
	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/health/service"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	service service.HealthService
}

func NewHealthHandler(service service.HealthService) *HealthHandler {
	return &HealthHandler{
		service: service,
	}
}

func (h *HealthHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", h.Health)
	r.GET("/ready", h.Readiness)
	r.GET("/live", h.Liveness)
}

// Health returns detailed health information including all checks
func (h *HealthHandler) Health(c *gin.Context) {
	health := h.service.GetHealth()

	statusCode := http.StatusOK
	switch health.Status {
	case domain.StatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case domain.StatusDegraded:
		statusCode = http.StatusOK // Still operational but degraded
	}

	c.JSON(statusCode, health)
}

// Readiness indicates if the service is ready to accept traffic
func (h *HealthHandler) Readiness(c *gin.Context) {
	readiness := h.service.GetReadiness()

	statusCode := http.StatusOK
	if readiness.Status != domain.StatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, readiness)
}

// Liveness is a simple check to see if the service is alive
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	})
}
