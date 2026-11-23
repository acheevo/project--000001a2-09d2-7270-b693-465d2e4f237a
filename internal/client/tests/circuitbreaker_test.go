package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	settings := client.Settings{
		Name:        "test-circuit-breaker",
		MaxRequests: 3,
		Interval:    100 * time.Millisecond,
		Timeout:     200 * time.Millisecond,
		ReadyToTrip: func(counts client.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	}

	cb := client.NewCircuitBreaker(settings)

	// Test initial state is closed
	assert.Equal(t, client.StateClosed, cb.State())

	// Test successful requests
	result, err := cb.Execute(func() (interface{}, error) {
		return "success", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, client.StateClosed, cb.State())

	// Test failures to trigger open state
	for i := 0; i < 2; i++ {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, errors.New("failure")
		})
		assert.Error(t, err)
	}

	// Circuit should now be open
	assert.Equal(t, client.StateOpen, cb.State())

	// Test requests are rejected when open
	_, err = cb.Execute(func() (interface{}, error) {
		return "should not execute", nil
	})
	assert.Error(t, err)
	assert.Equal(t, client.ErrCircuitOpen, err)

	// Wait for timeout to transition to half-open
	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, client.StateHalfOpen, cb.State())

	// Test successful request closes the circuit
	result, err = cb.Execute(func() (interface{}, error) {
		return "success after recovery", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "success after recovery", result)
	assert.Equal(t, client.StateClosed, cb.State())
}

func TestCircuitBreakerCounts(t *testing.T) {
	settings := client.Settings{
		Name:        "test-counts",
		MaxRequests: 5,
		Interval:    time.Second,
		Timeout:     time.Second,
	}

	cb := client.NewCircuitBreaker(settings)

	// Execute some successful requests
	for i := 0; i < 3; i++ {
		cb.Execute(func() (interface{}, error) {
			return "success", nil
		})
	}

	counts := cb.Counts()
	assert.Equal(t, uint32(3), counts.Requests)
	assert.Equal(t, uint32(3), counts.TotalSuccesses)
	assert.Equal(t, uint32(0), counts.TotalFailures)
	assert.Equal(t, uint32(3), counts.ConsecutiveSuccesses)

	// Execute a failure
	cb.Execute(func() (interface{}, error) {
		return nil, errors.New("failure")
	})

	counts = cb.Counts()
	assert.Equal(t, uint32(4), counts.Requests)
	assert.Equal(t, uint32(3), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.TotalFailures)
	assert.Equal(t, uint32(0), counts.ConsecutiveSuccesses)
	assert.Equal(t, uint32(1), counts.ConsecutiveFailures)
}
