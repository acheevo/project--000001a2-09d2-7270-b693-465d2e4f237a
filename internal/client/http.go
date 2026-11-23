package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/google/uuid"
)

type HTTPClient struct {
	client         *http.Client
	circuitBreaker *CircuitBreaker
	jwtService     service.JWTService
	serviceName    string
	serviceID      string
	version        string
	environment    string
	logger         *slog.Logger
}

type HTTPClientConfig struct {
	ServiceName string
	ServiceID   string
	Version     string
	Environment string
	Timeout     time.Duration
	JWTService  service.JWTService
	Logger      *slog.Logger
}

func NewHTTPClient(config HTTPClientConfig) *HTTPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Circuit breaker settings
	cbSettings := Settings{
		Name:        fmt.Sprintf("%s-circuit-breaker", config.ServiceName),
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from State, to State) {
			if config.Logger != nil {
				config.Logger.Info("circuit breaker state changed",
					"name", name,
					"from", from.String(),
					"to", to.String())
			}
		},
		IsSuccessful: func(err error) bool {
			return err == nil
		},
	}

	return &HTTPClient{
		client:         httpClient,
		circuitBreaker: NewCircuitBreaker(cbSettings),
		jwtService:     config.JWTService,
		serviceName:    config.ServiceName,
		serviceID:      config.ServiceID,
		version:        config.Version,
		environment:    config.Environment,
		logger:         config.Logger,
	}
}

func (c *HTTPClient) Do(ctx context.Context, method, url string, body interface{},
	headers map[string]string,
) (*http.Response, error) {
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doRequest(ctx, method, url, body, headers)
	})
	if err != nil {
		return nil, err
	}

	response, ok := result.(*http.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	return response, nil
}

func (c *HTTPClient) doRequest(ctx context.Context, method, url string, body interface{},
	headers map[string]string,
) (*http.Response, error) {
	var bodyReader io.Reader

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add service-to-service JWT token
	token, err := c.jwtService.GenerateServiceToken(c.serviceName, c.serviceID, c.version, c.environment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", c.serviceName, c.version))

	// Add X-Request-ID for tracing
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			req.Header.Set("X-Request-ID", id)
		}
	} else {
		req.Header.Set("X-Request-ID", uuid.New().String())
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	// Log the request
	if c.logger != nil {
		logFields := []interface{}{
			"method", method,
			"url", url,
			"duration", duration,
			"circuit_breaker_state", c.circuitBreaker.State().String(),
		}

		if err != nil {
			logFields = append(logFields, "error", err)
			c.logger.Error("HTTP request failed", logFields...)
		} else {
			logFields = append(logFields, "status", resp.StatusCode)
			if resp.StatusCode >= 400 {
				c.logger.Warn("HTTP request returned error status", logFields...)
			} else {
				c.logger.Debug("HTTP request completed", logFields...)
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Consider 5xx status codes as failures for circuit breaker
	if resp.StatusCode >= 500 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return resp, nil
}

func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, url, nil, headers)
}

func (c *HTTPClient) Post(ctx context.Context, url string, body interface{},
	headers map[string]string,
) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, url, body, headers)
}

func (c *HTTPClient) Put(ctx context.Context, url string, body interface{},
	headers map[string]string,
) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, url, body, headers)
}

func (c *HTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, url, nil, headers)
}

func (c *HTTPClient) CircuitBreakerState() State {
	return c.circuitBreaker.State()
}

func (c *HTTPClient) CircuitBreakerCounts() Counts {
	return c.circuitBreaker.Counts()
}
