package loadbalancer_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-gateway/internal/loadbalancer"
)

func TestNewRoundRobin(t *testing.T) {
	t.Run("valid targets", func(t *testing.T) {
		rr, err := loadbalancer.NewRoundRobin([]string{
			"http://localhost:8081",
			"http://localhost:8082",
		})
		if err != nil {
			t.Fatalf("NewRoundRobin returned error: %v", err)
		}
		if rr.ServerCount() != 2 {
			t.Errorf("ServerCount = %d, want 2", rr.ServerCount())
		}
	})

	t.Run("invalid target", func(t *testing.T) {
		_, err := loadbalancer.NewRoundRobin([]string{"://invalid"})
		if err == nil {
			t.Error("expected error for invalid URL, got nil")
		}
	})
}

func TestRoundRobin_NextServer(t *testing.T) {
	rr, err := loadbalancer.NewRoundRobin([]string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	})
	if err != nil {
		t.Fatal(err)
	}

	counts := make(map[string]int)
	for i := 0; i < 6; i++ {
		s := rr.NextServer()
		if s == nil {
			t.Fatal("NextServer returned nil")
		}
		counts[s.URL.String()]++
	}

	for url, count := range counts {
		if count != 2 {
			t.Errorf("server %s selected %d times, want 2", url, count)
		}
	}
}

func TestRoundRobin_SkipsDeadServers(t *testing.T) {
	rr, err := loadbalancer.NewRoundRobin([]string{
		"http://localhost:8081",
		"http://localhost:8082",
	})
	if err != nil {
		t.Fatal(err)
	}

	rr.SetServerAlive(0, false)

	for i := 0; i < 5; i++ {
		s := rr.NextServer()
		if s == nil {
			t.Fatal("NextServer returned nil")
		}
		if s.URL.String() != "http://localhost:8082" {
			t.Errorf("got %s, want http://localhost:8082", s.URL.String())
		}
	}
}

func TestRoundRobin_NoHealthyBackends(t *testing.T) {
	rr, err := loadbalancer.NewRoundRobin([]string{"http://localhost:8081"})
	if err != nil {
		t.Fatal(err)
	}
	rr.SetServerAlive(0, false)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestRoundRobin_FailoverOn5xx(t *testing.T) {
	var attempts int

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv1.Close)

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv2.Close)

	rr, err := loadbalancer.NewRoundRobin([]string{srv1.URL, srv2.URL})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr.ServeHTTP(rec, req)
	}

	if rr.IsServerAlive(0) {
		t.Error("server 1 should be marked dead after 3 failures")
	}
	if attempts < 4 {
		t.Errorf("expected at least 4 attempts, got %d", attempts)
	}
}

func TestRoundRobin_MarksServerUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	rr, err := loadbalancer.NewRoundRobin([]string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr.ServeHTTP(rec, req)
	}

	if rr.IsServerAlive(0) {
		t.Error("server should be marked dead after 3 consecutive failures")
	}
}

func TestRoundRobin_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	rr, err := loadbalancer.NewRoundRobin([]string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	rr.SetServerAlive(0, false)
	rr.StartHealthCheck()

	<-time.After(6 * time.Second)

	if !rr.IsServerAlive(0) {
		t.Error("server should be marked alive by health check")
	}
}

func TestRoundRobin_CircuitBreakerIntegration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	rr, err := loadbalancer.NewRoundRobin([]string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr.ServeHTTP(rec, req)
		if i < 4 {
			rr.ResetFailureCount(0)
			rr.SetServerAlive(0, true)
		}
	}

	if got := rr.CircuitBreakerState(0); got.String() != "open" {
		t.Errorf("circuit breaker state = %q, want %q", got.String(), "open")
	}
}
