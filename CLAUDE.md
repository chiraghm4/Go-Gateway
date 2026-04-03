# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go run cmd/main.go          # Start the gateway on :8080
go build -o gateway cmd/main.go  # Build binary
```

## Architecture

API Gateway in Go that routes requests to backend services with:

- **Reverse Proxy** (`internal/proxy/proxy.go`) - Single host reverse proxy using `httputil.NewSingleHostReverseProxy`
- **Load Balancer** (`internal/loadbalancer/roundrobin.go`) - Round-robin distribution across multiple backend servers with periodic health checks (5s interval)
- **Rate Limiter** (`internal/middleware/ratelimiter.go`) - Token bucket algorithm per IP (configurable capacity/refill rate)
- **Middleware Chain** (`internal/middleware/chain.go`) - Functional composition: `Chain(handler, middleware1, middleware2, ...)` wraps as `middleware1(middleware2(handler))`

## Request Flow

```
Request → Rate Limiter → Logging → Load Balancer/Proxy → Backend
```

Routes configured in `internal/router/router.go`:
- `/users` → Load-balanced across 3 servers (8081, 8082, 8083)
- `/orders` → Single backend (8082)

## Key Configuration

Rate limiter defaults: 250 tokens capacity, 250 tokens/sec refill rate
