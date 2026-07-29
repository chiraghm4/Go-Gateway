# AGENTS.md

## Commands

```bash
go run cmd/main.go              # Start gateway on :8080 (reads ../gateway-config.yaml)
go build -o gateway cmd/main.go # Build binary
go test ./...                   # Run all tests
go test ./internal/loadbalancer/ -run TestName   # Run single test
```

## Critical Setup

- **Config path is relative**: `cmd/main.go` reads `../gateway-config.yaml` — run from repo root
- **Backend services required**: Ports 8081, 8082, 8083 must be running (see `dummyServices/`)
- **Gateway config**: See `gateway-config1.yaml` for format (upstreams + endpoints)

## Architecture

Client → Gateway (8080) → Backend Services (8081-8083)

```
Request → LoggingMiddleware → RateLimiter → LoadBalancer/Proxy → Backend
```

### Key Files

| Component | Path |
|-----------|------|
| Entry point | `cmd/main.go` |
| Routes | `internal/router/router.go` |
| Load balancer | `internal/loadbalancer/roundrobin.go` |
| Circuit breaker | `internal/circuitbreaker/circuitbreaker.go` |
| Middleware chain | `internal/middleware/chain.go` |
| Reverse proxy | `internal/proxy/proxy.go` |

## Gotchas

- **Tests are in separate package**: `test/` directory, not `_test.go` files alongside code
- **Circuit breaker**: Opens after 5 failures, half-open after 30s
- **Rate limiter**: 250 requests/burst per IP
- **Health checks**: Background goroutine polls backends every 5s
- **.env file contains API keys** — never commit
