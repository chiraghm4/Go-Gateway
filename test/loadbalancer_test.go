package test

import (
	"api-gateway/internal/loadbalancer"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRoundRobin(t *testing.T) {
	targets := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	rr, err := loadbalancer.NewRoundRobin(targets)
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	if rr.ServerCount() != 2 {
		t.Errorf("Expected 2 servers, got %d", rr.ServerCount())
	}
}

func TestNewRoundRobin_InvalidTarget(t *testing.T) {
	invalidTargets := []string{
		"://invalid-url",
	}

	_, err := loadbalancer.NewRoundRobin(invalidTargets)
	if err == nil {
		t.Error("Expected error for invalid target, got nil")
	}
}

func TestRoundRobin_NextServer(t *testing.T) {
	targets := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	rr, err := loadbalancer.NewRoundRobin(targets)
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Test round robin distribution
	servers := make(map[string]int)
	for i := 0; i < 6; i++ {
		server := rr.NextServer()
		servers[server.URL.String()]++
	}

	// Each server should be selected twice
	for url, count := range servers {
		if count != 2 {
			t.Errorf("Server %s selected %d times, expected 2", url, count)
		}
	}
}

func TestRoundRobin_SkipsDeadServers(t *testing.T) {
	targets := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	rr, err := loadbalancer.NewRoundRobin(targets)
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Kill first server
	rr.SetServerAlive(0, false)

	// Next server should be the alive one
	server := rr.NextServer()
	if server.URL.String() != "http://localhost:8082" {
		t.Errorf("Expected server 8082, got %s", server.URL.String())
	}
}

func TestRoundRobin_NoHealthyBackends(t *testing.T) {
	targets := []string{
		"http://localhost:8081",
	}

	rr, err := loadbalancer.NewRoundRobin(targets)
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Kill all servers
	rr.SetServerAlive(0, false)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	rr.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}
}

func TestRoundRobin_FailoverOn5xx(t *testing.T) {
	attemptCount := 0

	// Create two test servers - first always returns 500, second returns 200
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	rr, err := loadbalancer.NewRoundRobin([]string{server1.URL, server2.URL})
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Make multiple requests - should failover to server2
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, req)
	}

	// After 3 failures, server1 should be marked dead
	if rr.IsServerAlive(0) {
		t.Error("Server1 should be marked dead after 3 failures")
	}

	// All subsequent requests should go to server2
	if attemptCount < 4 {
		t.Errorf("Expected at least 4 attempts (3 failures + failover), got %d", attemptCount)
	}
}

func TestRoundRobin_MarksServerUnhealthy(t *testing.T) {
	// Create a test server that always returns 500
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	rr, err := loadbalancer.NewRoundRobin([]string{testServer.URL})
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Make 3 failing requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, req)
	}

	// Server should be marked dead after 3 failures
	if rr.IsServerAlive(0) {
		t.Error("Server should be marked dead after 3 consecutive failures")
	}
}

func TestRoundRobin_HealthCheck(t *testing.T) {
	// Create a test server with health endpoint
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()

	rr, err := loadbalancer.NewRoundRobin([]string{testServer.URL})
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Mark server as dead
	rr.SetServerAlive(0, false)

	// Start health check
	rr.StartHealthCheck()

	// Wait for health check to run
	time.Sleep(6 * time.Second)

	// Server should be alive now
	if !rr.IsServerAlive(0) {
		t.Error("Server should be marked alive by health check")
	}
}

func TestRoundRobin_CircuitBreakerIntegration(t *testing.T) {
	// Create a single test server that returns 500
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer testServer.Close()

	rr, err := loadbalancer.NewRoundRobin([]string{testServer.URL})
	if err != nil {
		t.Fatalf("Failed to create RoundRobin: %v", err)
	}

	// Prevent server from being marked dead so circuit breaker gets all the failures
	// We need 5 failures to trip the circuit breaker, but server dies after 3
	// So we manually reset failure count and keep the server alive
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, req)
		// Reset failure count after each request to prevent server being marked dead
		// This lets circuit breaker accumulate its own failures
		if i < 4 {
			rr.ResetFailureCount(0)
			rr.SetServerAlive(0, true)
		}
	}

	// Circuit should be open now
	if rr.CircuitBreakerState(0).String() != "open" {
		t.Errorf("Expected circuit breaker to be open, got %v", rr.CircuitBreakerState(0))
	}
}
