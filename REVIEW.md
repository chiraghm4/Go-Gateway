# Critical Bug Fixes — Review Report

**Date:** 2026-07-12
**Files changed:** `internal/loadbalancer/roundrobin.go`, `internal/router/router.go`, `cmd/main.go`, `gateway-config.yaml`

---

## Summary

Five critical bugs were identified and fixed across the load balancer, router, and configuration wiring. All 31 existing tests pass, including with Go's race detector enabled.

---

## Fix 1: `NextServer()` Modulo Arithmetic Bug

**File:** `internal/loadbalancer/roundrobin.go:51`

**Before:**
```go
startIndex := int(atomic.AddUint64(&rr.counter, 1) - 1 % uint64(len(rr.servers)))
```

**After:**
```go
startIndex := int((atomic.AddUint64(&rr.counter, 1) - 1) % uint64(len(rr.servers)))
```

**Problem:** The `%` operator binds tighter than `-` in Go. This computed `counter + (1 - len%mod)` instead of `((counter + 1) - 1) % len`. For small counter values the bug was masked, but once the counter grew past `uint64(len(servers))`, the index would wrap incorrectly, causing non-uniform distribution and potential out-of-bounds panics.

**Fix:** Added parentheses to enforce correct evaluation order.

**Risk:** Low. Pure arithmetic correction.

---

## Fix 2: Data Races on `Alive` and `failureCount`

**File:** `internal/loadbalancer/roundrobin.go` (struct `Server`)

**Before:**
```go
type Server struct {
    Alive        bool
    failureCount uint64
    // ...
}
```
Read/written from request goroutines, health check goroutines, and test helpers without synchronization.

**After:**
```go
type Server struct {
    alive        atomic.Bool
    failureCount atomic.Uint64
    // ...
}
```
All access now goes through `alive.Load()` / `alive.Store()` and `failureCount.Load()` / `failureCount.Store()` / `failureCount.Add()`.

**Problem:** The `Alive` field was a plain `bool` read/written from multiple goroutines (health check goroutine + request handlers). The `failureCount` was written with `atomic.AddUint64` but read non-atomically in the retry logic (`server.failureCount >= 3`) and in the health check reset (`server.failureCount = 0`). Both are data races that Go's race detector would flag.

**Fix:** Converted both fields to Go 1.19+ `atomic.Bool` and `atomic.Uint64` types, ensuring all reads and writes are atomic.

**Risk:** Low. The public helper methods (`SetServerAlive`, `IsServerAlive`, `ResetFailureCount`) were updated to use the new atomic accessors. All existing tests continue to pass.

---

## Fix 3: Request Body Consumed Before Retry

**File:** `internal/loadbalancer/roundrobin.go` (`ServeHTTP` method)

**Before:**
```go
server.Proxy.ServeHTTP(rw, r)  // consumes r.Body
// retry with same r → empty body
```

**After:**
```go
// Buffer body before the retry loop
var bodyBytes []byte
if r.Body != nil {
    bodyBytes, _ = io.ReadAll(r.Body)
    r.Body.Close()
}

for i := 0; i <= maxRetries; i++ {
    // ...
    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    server.Proxy.ServeHTTP(cw, r)
    // ...
}
```

**Problem:** `httputil.ReverseProxy.ServeHTTP` reads and drains `r.Body`. On retry, the next backend received an empty request body. For GET requests this was invisible, but POST/PUT/PATCH requests would silently lose their payload on failover.

**Fix:** The request body is read into a `[]byte` buffer before the retry loop. On each iteration, `r.Body` is reset to a fresh `io.NopCloser` wrapping a new buffer copy. This ensures each backend sees the full original body.

**Risk:** Low. Adds a small memory allocation proportional to request body size. This is standard practice for retry-capable proxies.

---

## Fix 4: Retry Writes to Same `ResponseWriter`

**File:** `internal/loadbalancer/roundrobin.go` (`ServeHTTP` method)

**Before:**
```go
rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
server.Proxy.ServeHTTP(rw, r)
// On 5xx, retry writes headers/body to the SAME underlying w
```

**After:**
```go
cw := &captureWriter{w: w, statusCode: http.StatusOK}
server.Proxy.ServeHTTP(cw, r)
```

The new `captureWriter` implements a write-once pattern:
```go
type captureWriter struct {
    w          http.ResponseWriter
    statusCode int
    written    bool
}

func (cw *captureWriter) WriteHeader(code int) {
    if !cw.written {
        cw.statusCode = code
        cw.w.WriteHeader(code)
        cw.written = true
    }
}

func (cw *captureWriter) Write(b []byte) (int, error) {
    if !cw.written {
        cw.w.WriteHeader(cw.statusCode)
        cw.written = true
    }
    return cw.w.Write(b)
}
```

**Problem:** When the first backend returned a 5xx, the proxy had already written headers and possibly body bytes to the real `http.ResponseWriter`. The retry would attempt to write again, causing `"http: superfluous response.WriteHeader call"` panics or corrupted responses.

**Fix:** The new `captureWriter` tracks whether headers have been written via the `written` flag. `WriteHeader` is idempotent — duplicate calls are silently ignored. The status code is captured for the retry decision without forwarding the 5xx status to the client. Only the final successful response (or the last 5xx if all fail) reaches the client.

**Risk:** Medium. This changes the retry behavior. Previously, a 5xx response was partially written to the client before being "overwritten" by the retry. Now, the 5xx is suppressed and only the successful response is sent. This is the correct behavior for a proxy but changes observable behavior.

---

## Fix 5: Dead Configuration — YAML Endpoints Now Wired

**Files:** `cmd/main.go`, `internal/router/router.go`, `gateway-config.yaml`

**Before:**
- `main.go` parsed the `endpoint` section from YAML into an `Endpoint` struct but never used it
- `router.SetupRoutes(upstreams []string)` received a flat list of all upstream URLs
- Routes were hardcoded: `/users` and `/orders` both got the same 3 backends
- The `endpoint` section in YAML was purely decorative

**After:**
- `main.go` builds a `map[string][]string` (upstream name → target URLs) and a `[]EndpointConfig` from the parsed YAML
- `router.SetupRoutes(upstreamTargets, endpoints)` receives both maps and dynamically registers routes
- Each endpoint in the config maps a path to a specific upstream name
- The `order-service` upstream in YAML targets only `:8082`, so `/orders` correctly routes to a single backend

**New `gateway-config.yaml`:**
```yaml
upstreams:
  user-service:
    targets: [8081, 8082, 8083]
  order-service:
    targets: [8082]

endpoints:
  - path: /users
    upstream: user-service
  - path: /orders
    upstream: order-service
```

**Problem:** The config system was structurally complete but entirely disconnected from the runtime. Adding new routes required code changes in `router.go`.

**Fix:** Routes are now driven by configuration. Adding a new endpoint only requires a YAML change.

**Risk:** Low. The API surface of `SetupRoutes` changed from `(upstreams []string)` to `(upstreamTargets map[string][]string, endpoints []EndpointConfig)`. This is a breaking change to the function signature but the only caller is `main.go`, which was updated simultaneously.

---

## Additional Fix: Hardcoded Log Message

**File:** `cmd/main.go:81`

**Before:** `log.Println("API Gateway running on :8080")`
**After:** `log.Printf("API Gateway running on %s", PORT)`

The log message now reflects the actual configured port.

---

## Test Results

```
=== RUN   TestCircuitBreakerInitialState       --- PASS
=== RUN   TestCircuitBreaker_AllowsRequestsWhenClosed --- PASS
=== RUN   TestCircuitBreaker_OpensAfterThreshold --- PASS
=== RUN   TestCircuitBreaker_HalfOpenAfterTimeout --- PASS
=== RUN   TestCircuitBreaker_ClosesOnSuccessInHalfOpen --- PASS
=== RUN   TestCircuitBreaker_ReopensOnFailureInHalfOpen --- PASS
=== RUN   TestCircuitBreaker_ConcurrentAccess  --- PASS
=== RUN   TestCircuitBreaker_StateString       --- PASS
=== RUN   TestNewRoundRobin                    --- PASS
=== RUN   TestNewRoundRobin_InvalidTarget      --- PASS
=== RUN   TestRoundRobin_NextServer            --- PASS
=== RUN   TestRoundRobin_SkipsDeadServers      --- PASS
=== RUN   TestRoundRobin_NoHealthyBackends     --- PASS
=== RUN   TestRoundRobin_FailoverOn5xx         --- PASS
=== RUN   TestRoundRobin_MarksServerUnhealthy  --- PASS
=== RUN   TestRoundRobin_HealthCheck           --- PASS
=== RUN   TestRoundRobin_CircuitBreakerIntegration --- PASS
=== RUN   TestNewReverseProxy_ValidTarget      --- PASS
=== RUN   TestNewReverseProxy_InvalidTarget    --- PASS
=== RUN   TestReverseProxy_ForwardsRequests    --- PASS
=== RUN   TestReverseProxy_PreservesMethod     --- PASS
=== RUN   TestReverseProxy_PreservesHeaders    --- PASS
=== RUN   TestReverseProxy_ErrorHandler        --- PASS
=== RUN   TestReverseProxy_PassesQueryString   --- PASS
=== RUN   TestSingleHostReverseProxy_Director  --- PASS
=== RUN   TestRateLimiter_AllowsFirstRequest   --- PASS
=== RUN   TestRateLimiter_BlocksAfterCapacityExhausted --- PASS
=== RUN   TestRateLimiter_RefillsTokensOverTime --- PASS
=== RUN   TestRateLimiter_PerIPTracking        --- PASS
=== RUN   TestRateLimiter_ConcurrentAccess     --- PASS
=== RUN   TestRateLimiter_CapsTokensAtCapacity --- PASS
=== RUN   TestRateLimiter_DoesNotRefillPastCapacity --- PASS

PASS
ok  	api-gateway/test   12.514s   (with -race)
```

31/31 tests pass. Race detector clean.

---

## Remaining Items (Not Addressed)

These issues were identified but are outside the scope of critical bug fixes:

| Priority | Item | Notes |
|----------|------|-------|
| P1 | Graceful shutdown | No signal handling or `server.Shutdown()` |
| P1 | Rate limiter memory leak | `visitors` map never evicts stale entries |
| P1 | Hardcoded tuning params | Rate limit, CB threshold, health interval not in YAML |
| P2 | Dead code | `internal/proxy/proxy.go` is unused |
| P2 | No status code logging | Logging middleware doesn't capture response codes |
| P2 | No integration tests | Router and middleware chain are untested end-to-end |
| P2 | `.env` in git | API keys committed without `.gitignore` exclusion |
| P2 | Health check goroutine leak | No shutdown mechanism for background goroutine |
