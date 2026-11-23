package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// HTTP Server Configuration
	// Server listen address and port
	HTTPAddr string `envconfig:"HTTP_ADDR" default:":8080"`
	// Maximum duration for reading the entire request
	ReadTimeout string `envconfig:"READ_TIMEOUT" default:"15s"`
	// Maximum duration before timing out writes of the response
	WriteTimeout string `envconfig:"WRITE_TIMEOUT" default:"15s"`
	// Maximum time to wait for the next request when keep-alives are enabled
	IdleTimeout string `envconfig:"IDLE_TIMEOUT" default:"60s"`

	// Database Configuration (optional for some microservices)
	// Database host address
	DatabaseURL string `envconfig:"DATABASE_URL"`
	DBHost      string `envconfig:"DB_HOST" default:"localhost"`
	// Database port number
	DBPort string `envconfig:"DB_PORT" default:"5432"`
	// Database username for authentication
	DBUser string `envconfig:"DB_USER" default:"postgres"`
	// Database password for authentication
	DBPassword string `envconfig:"DB_PASSWORD" default:"postgres"`
	// Database name to connect to
	DBName string `envconfig:"DB_NAME" default:"api_template"`
	// SSL mode for database connection (disable, require, verify-ca, verify-full)
	DBSSLMode string `envconfig:"DB_SSL_MODE" default:"disable"`
	// Maximum number of idle connections in the pool
	DBMaxIdleConns int `envconfig:"DB_MAX_IDLE_CONNS" default:"10"`
	// Maximum number of open connections to the database
	DBMaxOpenConns int `envconfig:"DB_MAX_OPEN_CONNS" default:"100"`
	// Maximum amount of time a connection may be reused
	DBConnMaxLifetime string `envconfig:"DB_CONN_MAX_LIFETIME" default:"1h"`

	// Service Identity
	// Name of this service for identification and logging
	ServiceName string `envconfig:"SERVICE_NAME" default:"nimbus-rbac-stage"`
	// Version of this service for tracking and compatibility
	ServiceVersion string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	// Unique identifier for this service instance (auto-generated if empty)
	ServiceID string `envconfig:"SERVICE_ID" default:""`

	// Logging Configuration
	// Log level (debug, info, warn, error)
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	// Application environment (development, staging, production)
	Environment string `envconfig:"ENVIRONMENT" default:"development"`

	// Security Configuration
	// Secret key for signing JWT tokens
	JWTSecret string `envconfig:"JWT_SECRET" default:"development-jwt-secret"`
	// JWT token expiration duration
	JWTExpiration string `envconfig:"JWT_EXPIRATION" default:"24h"`

	// Observability
	// Enable Prometheus metrics collection and /metrics endpoint
	MetricsEnabled bool `envconfig:"METRICS_ENABLED" default:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.IsProduction() && len(strings.TrimSpace(cfg.JWTSecret)) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes in production")
	}
	return &cfg, nil
}

func (c *Config) DatabaseDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	host := c.DBHost
	if host == "localhost" {
		host = "127.0.0.1"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) JWTExpirationDuration() time.Duration {
	duration, err := time.ParseDuration(c.JWTExpiration)
	if err != nil {
		return 24 * time.Hour // fallback to 24 hours
	}
	return duration
}

func (c *Config) ReadTimeoutDuration() time.Duration {
	duration, err := time.ParseDuration(c.ReadTimeout)
	if err != nil {
		return 15 * time.Second
	}
	return duration
}

func (c *Config) WriteTimeoutDuration() time.Duration {
	duration, err := time.ParseDuration(c.WriteTimeout)
	if err != nil {
		return 15 * time.Second
	}
	return duration
}

func (c *Config) IdleTimeoutDuration() time.Duration {
	duration, err := time.ParseDuration(c.IdleTimeout)
	if err != nil {
		return 60 * time.Second
	}
	return duration
}

func (c *Config) DBConnMaxLifetimeDuration() time.Duration {
	duration, err := time.ParseDuration(c.DBConnMaxLifetime)
	if err != nil {
		return time.Hour
	}
	return duration
}

func (c *Config) GetServiceID() string {
	if c.ServiceID == "" {
		// Generate a unique service ID if not provided
		return fmt.Sprintf("%s-%d", c.ServiceName, time.Now().Unix())
	}
	return c.ServiceID
}
