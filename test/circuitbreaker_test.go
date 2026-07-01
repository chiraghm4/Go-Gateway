package test

import (
	"api-gateway/internal/circuitbreaker"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreakerInitialState(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)

	if cb.State() != circuitbreaker.StateClosed {
		t.Errorf("Expected initial state to be Closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_AllowsRequestsWhenClosed(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)

	for i := 0; i < 4; i++ {
		if !cb.Allow() {
			t.Errorf("Request %d should be allowed when circuit is closed", i+1)
		}
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)

	// Record failures up to threshold
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.State() != circuitbreaker.StateOpen {
		t.Errorf("Expected state to be Open after 5 failures, got %v", cb.State())
	}

	// Requests should be blocked when circuit is open
	if cb.Allow() {
		t.Error("Requests should not be allowed when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := circuitbreaker.New(5, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("Expected state to be Open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Next Allow() should transition to half-open and allow request
	if !cb.Allow() {
		t.Error("Request should be allowed after timeout transitions to half-open")
	}

	if cb.State() != circuitbreaker.StateHalfOpen {
		t.Errorf("Expected state to be HalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreaker_ClosesOnSuccessInHalfOpen(t *testing.T) {
	cb := circuitbreaker.New(5, 50*time.Millisecond)

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Wait for half-open
	time.Sleep(100 * time.Millisecond)
	cb.Allow() // Transition to half-open

	// Record success should close circuit
	cb.RecordSuccess()

	if cb.State() != circuitbreaker.StateClosed {
		t.Errorf("Expected state to be Closed after success, got %v", cb.State())
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	cb := circuitbreaker.New(5, 50*time.Millisecond)

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Wait for half-open
	time.Sleep(100 * time.Millisecond)
	cb.Allow() // Transition to half-open

	// Failure in half-open should reopen
	cb.RecordFailure()

	if cb.State() != circuitbreaker.StateOpen {
		t.Errorf("Expected state to be Open after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Simulate concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
			cb.RecordFailure()
		}()
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		t.Error("Concurrent access caused errors")
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state    circuitbreaker.State
		expected string
	}{
		{circuitbreaker.StateClosed, "closed"},
		{circuitbreaker.StateOpen, "open"},
		{circuitbreaker.StateHalfOpen, "half-open"},
		{99, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}
