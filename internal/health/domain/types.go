package domain

import "time"

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

type HealthResponse struct {
	Status    Status               `json:"status"`
	Service   ServiceInfo          `json:"service"`
	Timestamp time.Time            `json:"timestamp"`
	Uptime    time.Duration        `json:"uptime"`
	Checks    map[string]CheckInfo `json:"checks,omitempty"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Instance    string `json:"instance"`
}

type CheckInfo struct {
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

type ReadinessResponse struct {
	Status    Status               `json:"status"`
	Service   ServiceInfo          `json:"service"`
	Timestamp time.Time            `json:"timestamp"`
	Checks    map[string]CheckInfo `json:"checks"`
}

type Check interface {
	Name() string
	Check() CheckInfo
}
