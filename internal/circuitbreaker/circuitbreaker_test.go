package circuitbreaker_test

import (
	"sync"
	"testing"
	"time"

	"api-gateway/internal/circuitbreaker"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Errorf("initial state = %v, want %v", got, circuitbreaker.StateClosed)
	}
}

func TestCircuitBreaker_AllowsRequestsWhenClosed(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)
	for i := 0; i < 4; i++ {
		if !cb.Allow() {
			t.Fatal("circuit should allow requests when closed")
		}
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if got := cb.State(); got != circuitbreaker.StateOpen {
		t.Fatalf("state after 5 failures = %v, want %v", got, circuitbreaker.StateOpen)
	}
	if cb.Allow() {
		t.Error("circuit should block requests when open")
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cb := circuitbreaker.New(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	<-time.After(100 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("circuit should allow probe request after timeout")
	}
	if got := cb.State(); got != circuitbreaker.StateHalfOpen {
		t.Errorf("state after timeout = %v, want %v", got, circuitbreaker.StateHalfOpen)
	}
}

func TestCircuitBreaker_ClosesOnSuccessInHalfOpen(t *testing.T) {
	cb := circuitbreaker.New(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	<-time.After(100 * time.Millisecond)
	cb.Allow()
	cb.RecordSuccess()
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Errorf("state after success in half-open = %v, want %v", got, circuitbreaker.StateClosed)
	}
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	cb := circuitbreaker.New(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	<-time.After(100 * time.Millisecond)
	cb.Allow()
	cb.RecordFailure()
	if got := cb.State(); got != circuitbreaker.StateOpen {
		t.Errorf("state after failure in half-open = %v, want %v", got, circuitbreaker.StateOpen)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := circuitbreaker.New(5, 30*time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
			cb.RecordFailure()
		}()
	}
	wg.Wait()
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state circuitbreaker.State
		want  string
	}{
		{circuitbreaker.StateClosed, "closed"},
		{circuitbreaker.StateOpen, "open"},
		{circuitbreaker.StateHalfOpen, "half-open"},
		{99, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}
