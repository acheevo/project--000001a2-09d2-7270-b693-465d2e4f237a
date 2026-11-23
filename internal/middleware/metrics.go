package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	methodLabel = "method"
	pathLabel   = "path"
	statusLabel = "status"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "The total number of HTTP requests",
		},
		[]string{methodLabel, pathLabel, statusLabel},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "The HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{methodLabel, pathLabel, statusLabel},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "The current number of HTTP requests being served",
		},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "The HTTP request sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{methodLabel, pathLabel},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "The HTTP response sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{methodLabel, pathLabel, statusLabel},
	)
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment in-flight requests
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		method := metricMethod(c.Request.Method)
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		requestSize := computeApproximateRequestSize(c.Request)
		httpRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))

		// Process request
		c.Next()

		// Record metrics after request completion
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
		httpResponseSize.WithLabelValues(method, path, status).Observe(float64(c.Writer.Size()))
	}
}

func metricMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}

// computeApproximateRequestSize returns the approximate size of the request in bytes
func computeApproximateRequestSize(r *http.Request) int {
	size := 0
	if r.URL != nil {
		size += len(r.URL.Path)
	}

	size += len(r.Method)
	size += len(r.Proto)

	for name, values := range r.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}

	size += len(r.Host)

	if r.ContentLength != -1 {
		size += int(r.ContentLength)
	}

	return size
}
