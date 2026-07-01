# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go run cmd/main.go              # Start the API Gateway on :8080
go build -o gateway cmd/main.go # Build the binary
```

## Architecture

**Client → Gateway (8080) → Backend Services (8081-8083)**

The gateway is a reverse proxy with middleware chaining:

```
Request → LoggingMiddleware → RateLimiter → LoadBalancer/Proxy → Backend
```

### Core Components

- **`cmd/main.go`** - Entry point, creates HTTP server with 5s read/10s write timeouts
- **`internal/router/router.go`** - Route registration and middleware wiring
- **`internal/middleware/chain.go`** - Middleware chaining (wraps handlers right-to-left)
- **`internal/middleware/ratelimiter.go`** - Token Bucket algorithm (per-IP, mutex-protected)
- **`internal/middleware/logging.go`** - Logs method, path, and duration
- **`internal/loadbalancer/roundrobin.go`** - Round Robin with atomic counter + background health check (5s interval)
- **`internal/proxy/proxy.go`** - Wraps `httputil.NewSingleHostReverseProxy` with custom `ErrorHandler`
- **`internal/circuitbreaker/circuitbreaker.go`** - Three-state circuit breaker (Closed/Open/HalfOpen)
- **`dummyServices/userservices/*`** - Dummy backend services for testing

### Traffic Flow

- `/users` → Load balanced across 8081, 8082, 8083 (health-aware)
- `/orders` → Single backend on 8082

### Concurrency Model

- Atomic counters for thread-safe round robin (`atomic.AddUint64`)
- Mutex-guarded visitor map in rate limiter
- Atomic state/failure tracking in circuit breaker
- Background goroutine for health checks

### Resilience Patterns

**Retry Logic** (`internal/loadbalancer/roundrobin.go:ServeHTTP`):
- Retries on 5xx errors, fails over to next healthy server
- Marks server unhealthy after 3 consecutive failures

**Circuit Breaker** (`internal/circuitbreaker/circuitbreaker.go`):
- Threshold: 5 failures opens circuit
- Timeout: 30s before half-open state
- Half-open allows one test request to probe recovery

## Development practices
- Use comments wherever necessary. For complex code and for important concepts of golang and backend. 