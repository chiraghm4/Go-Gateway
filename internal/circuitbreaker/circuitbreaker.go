package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"time"
)

type State uint32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreaker struct {
	state        uint32
	failureCount uint64
	lastFailure  int64
	threshold    uint64
	timeout      time.Duration
	mu           sync.Mutex
}

func New(threshold uint64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     uint32(StateClosed),
		threshold: threshold,
		timeout:   timeout,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	state := cb.State()

	if state == StateClosed {
		return true
	}

	if state == StateOpen {
		if time.Since(time.Unix(atomic.LoadInt64(&cb.lastFailure), 0)) > cb.timeout {
			cb.mu.Lock()
			defer cb.mu.Unlock()
			if State(atomic.LoadUint32(&cb.state)) == StateOpen {
				atomic.StoreUint32(&cb.state, uint32(StateHalfOpen))
			}
			return true
		}
		return false
	}

	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreUint64(&cb.failureCount, 0)
	atomic.StoreUint32(&cb.state, uint32(StateClosed))
}

func (cb *CircuitBreaker) RecordFailure() {
	count := atomic.AddUint64(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailure, time.Now().Unix())

	if count >= cb.threshold {
		cb.mu.Lock()
		defer cb.mu.Unlock()
		if State(atomic.LoadUint32(&cb.state)) != StateOpen {
			atomic.StoreUint32(&cb.state, uint32(StateOpen))
		}
	}
}

func (cb *CircuitBreaker) State() State {
	return State(atomic.LoadUint32(&cb.state))
}
