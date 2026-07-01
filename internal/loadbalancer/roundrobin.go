package loadbalancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"api-gateway/internal/circuitbreaker"
)

type Server struct {
	URL          *url.URL
	Proxy        *httputil.ReverseProxy
	Alive        bool
	failureCount uint64
	cb           *circuitbreaker.CircuitBreaker
}

type RoundRobin struct {
	servers []*Server
	counter uint64
}

func NewRoundRobin(targets []string) (*RoundRobin, error) {
	var servers []*Server

	for _, t := range targets {
		u, err := url.Parse(t)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(u)

		servers = append(servers, &Server{
			URL:   u,
			Proxy: proxy,
			Alive: true,
			cb:    circuitbreaker.New(5, 30*time.Second),
		})
	}

	return &RoundRobin{
		servers: servers,
	}, nil
}

func (rr *RoundRobin) NextServer() *Server {
	startIndex := int(atomic.AddUint64(&rr.counter, 1) - 1 % uint64(len(rr.servers)))

	for i := 0; i < len(rr.servers); i++ {
		index := (startIndex + i) % len(rr.servers)
		server := rr.servers[index]

		if server.Alive {
			return server
		}
	}

	return nil // No healthy servers available
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	maxRetries := len(rr.servers) - 1

	for i := 0; i <= maxRetries; i++ {
		server := rr.NextServer()
		if server == nil {
			http.Error(w, "no healthy backends", http.StatusBadGateway)
			return
		}

		// Check circuit breaker before making request
		if !server.cb.Allow() {
			// Circuit is open, try next server
			continue
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		server.Proxy.ServeHTTP(rw, r)

		if rw.statusCode < 500 {
			server.failureCount = 0
			server.cb.RecordSuccess()
			return
		}

		// Record failure for both rate limiting and circuit breaker
		atomic.AddUint64(&server.failureCount, 1)
		if server.failureCount >= 3 {
			server.Alive = false
		}
		server.cb.RecordFailure()
	}

	http.Error(w, "all backends unavailable", http.StatusBadGateway)
}

func (rr *RoundRobin) StartHealthCheck() {
	go func() {
		for {
			for _, server := range rr.servers {
				resp, err := http.Get(server.URL.String() + "/health")

				if err != nil || resp.StatusCode != 200 {
					server.Alive = false
				} else {
					server.Alive = true
					server.failureCount = 0
				}

				if resp != nil {
					resp.Body.Close()
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

// ServerCount returns the number of servers (for testing)
func (rr *RoundRobin) ServerCount() int {
	return len(rr.servers)
}

// GetServer returns a server by index (for testing)
func (rr *RoundRobin) GetServer(index int) *Server {
	if index < 0 || index >= len(rr.servers) {
		return nil
	}
	return rr.servers[index]
}

// SetServerAlive sets a server's alive status (for testing)
func (rr *RoundRobin) SetServerAlive(index int, alive bool) {
	if index >= 0 && index < len(rr.servers) {
		rr.servers[index].Alive = alive
	}
}

// IsServerAlive returns a server's alive status (for testing)
func (rr *RoundRobin) IsServerAlive(index int) bool {
	if index >= 0 && index < len(rr.servers) {
		return rr.servers[index].Alive
	}
	return false
}

// CircuitBreakerState returns the circuit breaker state for a server (for testing)
func (rr *RoundRobin) CircuitBreakerState(index int) circuitbreaker.State {
	if index >= 0 && index < len(rr.servers) {
		return rr.servers[index].cb.State()
	}
	return circuitbreaker.StateClosed
}

// ResetFailureCount resets the failure count for a server (for testing)
func (rr *RoundRobin) ResetFailureCount(index int) {
	if index >= 0 && index < len(rr.servers) {
		rr.servers[index].failureCount = 0
	}
}
