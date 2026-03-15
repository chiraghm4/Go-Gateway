# Go API Gateway

A simple **API Gateway implemented in Golang**.

This project was built while learning Go to better understand how real backend infrastructure components like **reverse proxies, rate limiters, and load balancers** work.

The goal was to build a **practical system instead of just small tutorial examples**.

---

## Features

- Reverse proxy request forwarding
- Middleware chaining
- Logging middleware
- Token Bucket rate limiting
- Round Robin load balancing
- Backend health checks
- Concurrency using goroutines
- Atomic counters for safe concurrent access

---

## Architecture

Client → API Gateway → Backend Services

Example flow:

Client
|
API Gateway
|
├── users-service-1
├── users-service-2
└── users-service-3


The gateway distributes traffic between backend services using **Round Robin load balancing**.

---

## Project Structure

api-gateway/
│
├── cmd/
│ └── main.go
│
├── internal/
│ ├── loadbalancer/
│ │ └── roundrobin.go
│ │
│ ├── middleware/
│ │ ├── chain.go
│ │ ├── logging.go
│ │ └── ratelimiter.go
│ │
│ ├── proxy/
│ │ └── proxy.go
│ │
│ └── router/
│ └── router.go
│
└── go.mod


---

## Concepts Implemented

### Reverse Proxy
The gateway forwards incoming requests to backend services using Go's `httputil.ReverseProxy`.

### Middleware
Custom middleware chain implementation similar to frameworks like Gin or Express.

### Rate Limiting
Token Bucket algorithm to limit incoming request rate.

### Load Balancing
Round Robin strategy distributes requests between backend services.

### Health Checks
Background health checker detects unhealthy services and removes them from rotation.

### Concurrency
Uses:
- Goroutines
- Atomic counters
- Background workers

---

## Running the Project

### 1. Clone the repository

git clone https://github.com/chiraghm4/Go-Gateway.git

cd api-gateway


### 2. Start backend services

Example services running on different ports:

localhost:8081
localhost:8082
localhost:8083

Each service should expose:

/users
/health

### 3. Start the API Gateway

go run cmd/main.go

Gateway runs on:
http://localhost:8080


---

## Test the Gateway

Send requests:
curl http://localhost:8080/users

Requests will be distributed across backend services.

---

## Example

Responses should rotate between:
users-service-1
users-service-2
users-service-3

---

## Learning Notes

This project was built while **learning Golang** and exploring backend architecture concepts.

Some implementation ideas and learning guidance were taken with help from **ChatGPT**.

---

## Future Improvements

Possible improvements:

- Retry logic
- Circuit breaker
- Distributed rate limiting (Redis)
- Metrics (Prometheus)
- gRPC support
- Dynamic service discovery

---

