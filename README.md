# Go API Gateway

A config-driven API Gateway in Go with reverse proxy, rate limiting, round-robin load balancing, circuit breaker, and health checks.

## Configuration

The gateway is configured entirely via `gateway-config.yaml` (placed one directory above the binary — `../gateway-config.yaml` relative to `cmd/main.go`).

### Full Example

```yaml
gateway:
  port: 8080
  timeout_seconds: 30

upstreams:
  user-service:
    targets:
      - host: "localhost"
        target_port: 8081
        protocol: http
      - host: "localhost"
        target_port: 8082
        protocol: http
      - host: "localhost"
        target_port: 8083
        protocol: http

  order-service:
    targets:
      - host: "localhost"
        target_port: 8082
        protocol: http

endpoints:
  - path: /users
    method: GET
    upstream: user-service
  - path: /orders
    method: GET
    upstream: order-service
```

### Sections

#### `gateway`

| Field | Description |
|-------|-------------|
| `port` | Port the gateway listens on |
| `timeout_seconds` | Read/Write timeout for the HTTP server (split equally) |

#### `upstreams`

A map of named upstream service groups. Each key is an arbitrary name referenced by endpoints.

| Field | Description |
|-------|-------------|
| `targets` | List of backend servers under this upstream |

Each target:

| Field | Description |
|-------|-------------|
| `host` | Backend hostname or IP |
| `target_port` | Backend port |
| `protocol` | `http` or `https` |

#### `endpoints`

A list of routes mapping URL paths to upstreams.

| Field | Description |
|-------|-------------|
| `path` | URL path pattern (e.g. `/users`) |
| `method` | HTTP method (currently informational; all methods match) |
| `upstream` | References a key from `upstreams` |

## Architecture

```
Client → LoggingMiddleware → RateLimiter → LoadBalancer/Proxy → Backend
```

- **Rate limiter**: Token bucket, 250 requests/burst per IP
- **Load balancer**: Round-robin across healthy backends
- **Circuit breaker**: Opens after 5 consecutive failures, half-open after 30s
- **Health checks**: Polls `/health` on each backend every 5s; unhealthy servers are skipped

## Quick Start

```bash
# 1. Start backend services (instances on 8081, 8082, 8083)
go run .\dummyServices\userservices\userService.go

# 2. Start the gateway (from repo root)
go run .\cmd\main.go

# 3. Test
curl http://localhost:8080/users
```

Requests are distributed across backends in round-robin. Unhealthy backends are automatically removed from rotation.

## Project Structure

```
cmd/main.go              Entry point — reads config, sets up routes
internal/
  router/router.go       Route registration + middleware wiring
  middleware/            Logging, rate limiter, middleware chain
  loadbalancer/          Round Robin + health checks + circuit breaker
  circuitbreaker/        Circuit breaker (closed → open → half-open)
  proxy/                 Reverse proxy wrapper
test/                    Tests (circuit breaker, load balancer, proxy, rate limiter)
dummyServices/           Sample backend services for local testing
```
