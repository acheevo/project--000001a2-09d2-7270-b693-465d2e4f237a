package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/auth/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

func NewRateLimiter(rps float64, burst int, cleanup time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		cleanup:  cleanup,
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for key, limiter := range rl.limiters {
			// Remove limiters that haven't been used recently
			if limiter.TokensAt(time.Now()) == float64(rl.burst) {
				delete(rl.limiters, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit creates a rate limiting middleware
// rps: requests per second
// burst: maximum burst size
// cleanup: cleanup interval for unused limiters
func RateLimit(rps float64, burst int, cleanup time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst, cleanup)

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.getLimiter(key).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				errorKey:  "rate limit exceeded",
				"message": "too many requests from this IP",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByService creates a rate limiting middleware for service-to-service calls
func RateLimitByService(rps float64, burst int, cleanup time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst, cleanup)

	return func(c *gin.Context) {
		// Try to get service name from JWT claims
		var key string
		if serviceCtx, exists := c.Get(ServiceContextKey); exists {
			if claims, ok := serviceCtx.(*service.Claims); ok {
				key = claims.ServiceName
			}
		}

		// Fallback to IP if no service context
		if key == "" {
			key = c.ClientIP()
		}

		if !limiter.getLimiter(key).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				errorKey:  "rate limit exceeded",
				"message": "too many requests from service: " + key,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
