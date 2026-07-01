# Retry Logic and Circuit Breaker Implementation

## Summary

Added resilience patterns to the API Gateway: retry logic with failover and a circuit breaker to protect against cascading backend failures.

## Files Created

### `internal/circuitbreaker/circuitbreaker.go`

Three-state circuit breaker implementation:

- **States**: Closed (normal), Open (rejecting), HalfOpen (testing recovery)
- **Threshold**: Opens after 5 consecutive failures
- **Timeout**: 30 seconds before transitioning from Open to HalfOpen
- **Thread-safe**: Uses atomic operations for state and failure counting

## Files Modified

### `internal/loadbalancer/roundrobin.go`

**Changes:**
- Added `responseWriter` wrapper to capture HTTP status codes
- Modified `ServeHTTP` to retry failed requests on different backends
- Retry logic: Retries on 5xx errors, up to N-1 retries for N servers
- Failure tracking: Marks server unhealthy after 3 consecutive failures
- Reset failure count on successful response

**Before:**
```go
func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    server := rr.NextServer()
    server.Proxy.ServeHTTP(w, r)
}
```

**After:**
```go
func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    maxRetries := len(rr.servers) - 1
    for i := 0; i <= maxRetries; i++ {
        server := rr.NextServer()
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        server.Proxy.ServeHTTP(rw, r)
        
        if rw.statusCode < 500 {
            server.failureCount = 0
            return
        }
        
        atomic.AddUint64(&server.failureCount, 1)
        if server.failureCount >= 3 {
            server.Alive = false
        }
    }
    http.Error(w, "all backends unavailable", http.StatusBadGateway)
}
```

### `internal/proxy/proxy.go`

**Changes:**
- Added custom `ErrorHandler` to log errors and return consistent 502 response

**Before:**
```go
proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
return proxy, nil
```

**After:**
```go
proxy := httputil.NewSingleHostReverseProxy(parsedUrl)
proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
    log.Printf("Proxy error for %s: %v", r.URL.Path, err)
    http.Error(w, "backend unavailable", http.StatusBadGateway)
}
return proxy, nil
```

### `internal/router/router.go`

**Changes:**
- Removed unused `time` import
- Removed dead code comments (commented-out proxy configurations)

### `CLAUDE.md`

**Added:**
- Documentation for `internal/circuitbreaker/circuitbreaker.go`
- Updated `internal/proxy/proxy.go` description
- New "Resilience Patterns" section documenting retry logic and circuit breaker behavior

## How It Works

### Request Flow

1. Request arrives at gateway
2. Rate limiter checks token bucket
3. Load balancer selects backend via round-robin
4. Proxy forwards request to backend
5. On 5xx response: retry on next healthy backend
6. On connection error: `ErrorHandler` returns 502, retry triggers

### Circuit Breaker Flow

```
Closed ──(5 failures)──> Open ──(30s timeout)──> HalfOpen
   ^                                              │
   │         ┌──────────(success)─────────┐       │
   └─────────┴─────────(failure)──────────┴───────┘
```

### Retry Behavior

- `/users` route (3 backends): Up to 2 retries on failure
- `/orders` route (1 backend): No retry (single target)
- Only retries 5xx server errors, not 4xx client errors

## Testing

1. Start backends:
   ```bash
   go run dummyServices/userservices/us1.go &
   go run dummyServices/userservices/us2.go &
   go run dummyServices/userservices/us3.go &
   ```

2. Start gateway:
   ```bash
   go run cmd/main.go
   ```

3. Test retry: Kill one backend, verify requests redirect to healthy servers

4. Test circuit breaker: Cause repeated failures, verify circuit opens after 5 failures

5. Test recovery: Wait 30s, verify half-open state allows test request
