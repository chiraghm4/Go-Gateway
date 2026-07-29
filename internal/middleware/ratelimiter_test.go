package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"api-gateway/internal/middleware"
)

func TestRateLimiter_AllowsFirstRequest(t *testing.T) {
	rl := middleware.NewRateLimiter(10, 10)
	var served bool
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !served {
		t.Error("first request should be allowed")
	}
}

func TestRateLimiter_BlocksAfterCapacityExhausted(t *testing.T) {
	rl := middleware.NewRateLimiter(3, 1)
	var allowed int
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed++
	}))

	addr := "127.0.0.1:12345"
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	if allowed != 3 {
		t.Fatalf("allowed = %d, want 3", allowed)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiter_PerIPTracking(t *testing.T) {
	rl := middleware.NewRateLimiter(2, 1)
	var ip1Count, ip2Count int
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RemoteAddr {
		case "10.0.0.1:12345":
			ip1Count++
		default:
			ip2Count++
		}
	}))

	for i := 0; i < 2; i++ {
		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.RemoteAddr = "10.0.0.1:12345"
		h.ServeHTTP(httptest.NewRecorder(), req1)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.RemoteAddr = "10.0.0.2:12345"
		h.ServeHTTP(httptest.NewRecorder(), req2)
	}

	if ip1Count != 2 {
		t.Errorf("ip1 allowed = %d, want 2", ip1Count)
	}
	if ip2Count != 2 {
		t.Errorf("ip2 allowed = %d, want 2", ip2Count)
	}
}

func TestRateLimiter_RefillsTokensOverTime(t *testing.T) {
	rl := middleware.NewRateLimiter(2, 10)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	addr := "127.0.0.1:12345"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	<-time.After(150 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusTooManyRequests {
		t.Error("request should be allowed after token refill")
	}
}

func TestRateLimiter_CapsTokensAtCapacity(t *testing.T) {
	rl := middleware.NewRateLimiter(5, 100)
	var allowed int
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed++
	}))

	addr := "127.0.0.1:12345"
	<-time.After(50 * time.Millisecond)

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	if allowed != 5 {
		t.Errorf("allowed = %d, want 5", allowed)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := middleware.NewRateLimiter(100, 100)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			h.ServeHTTP(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()
}
