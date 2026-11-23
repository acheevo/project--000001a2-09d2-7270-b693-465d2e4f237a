package domain

import "time"

type InfoResponse struct {
	Service      ServiceInfo      `json:"service"`
	System       SystemInfo       `json:"system"`
	Dependencies DependencyStatus `json:"dependencies"`
	Timestamp    time.Time        `json:"timestamp"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Instance    string `json:"instance"`
	Uptime      string `json:"uptime"`
}

type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	Platform     string `json:"platform"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
}

type DependencyStatus struct {
	Database DatabaseStatus `json:"database"`
}

type DatabaseStatus struct {
	Status       string        `json:"status"`
	Reachable    bool          `json:"reachable"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
}
