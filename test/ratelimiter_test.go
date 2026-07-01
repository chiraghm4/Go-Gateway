package test

import (
	"api-gateway/internal/middleware"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowsFirstRequest(t *testing.T) {
	rl := middleware.NewRateLimiter(10, 10)

	var allowed bool
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !allowed {
		t.Error("First request should be allowed")
	}

	if w.Code == http.StatusTooManyRequests {
		t.Error("First request should not return 429")
	}
}

func TestRateLimiter_BlocksAfterCapacityExhausted(t *testing.T) {
	rl := middleware.NewRateLimiter(3, 1) // 3 tokens capacity, 1 token/sec refill

	allowedCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedCount++
	}))

	// Make 3 requests rapidly - should all be allowed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	if allowedCount != 3 {
		t.Errorf("Expected 3 allowed requests, got %d", allowedCount)
	}

	// 4th request should be blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 status, got %d", w.Code)
	}
}

func TestRateLimiter_RefillsTokensOverTime(t *testing.T) {
	rl := middleware.NewRateLimiter(2, 10) // 2 tokens, 10 tokens/sec refill

	allowedCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedCount++
	}))

	// Exhaust tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Wait for refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected request to be allowed after refill, got %d", w.Code)
	}
}

func TestRateLimiter_PerIPTracking(t *testing.T) {
	rl := middleware.NewRateLimiter(2, 1)

	allowedCount1 := 0
	allowedCount2 := 0

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RemoteAddr == "192.168.1.1:12345" {
			allowedCount1++
		} else {
			allowedCount2++
		}
	}))

	// New visitor gets first request free + (capacity-1) tokens = 2 total for IP1
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP2 should also get 2 requests (separate bucket)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	if allowedCount1 != 2 {
		t.Errorf("Expected 2 allowed for IP1, got %d", allowedCount1)
	}

	if allowedCount2 != 2 {
		t.Errorf("Expected 2 allowed for IP2, got %d", allowedCount2)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := middleware.NewRateLimiter(100, 100)

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		allowedCount++
		mu.Unlock()
	}))

	// Fire 200 concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}()
	}

	wg.Wait()

	// Should allow ~100 requests (capacity)
	if allowedCount > 105 || allowedCount < 95 {
		t.Errorf("Expected ~100 allowed requests, got %d", allowedCount)
	}
}

func TestRateLimiter_CapsTokensAtCapacity(t *testing.T) {
	rl := middleware.NewRateLimiter(5, 100) // 5 tokens, 100 tokens/sec

	allowedCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedCount++
	}))

	// Wait for tokens to accumulate (should cap at 5)
	time.Sleep(100 * time.Millisecond)

	// Make 6 requests rapidly
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Should allow exactly 5 (capacity cap)
	if allowedCount != 5 {
		t.Errorf("Expected 5 allowed (capped at capacity), got %d", allowedCount)
	}
}

func TestRateLimiter_DoesNotRefillPastCapacity(t *testing.T) {
	rl := middleware.NewRateLimiter(3, 100)

	allowedCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedCount++
	}))

	// First request is free (visitor created with capacity-1 tokens = 2)
	// Then we have 2 more tokens available
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Wait for refill
	time.Sleep(100 * time.Millisecond)

	// Should have ~1 more token from refill (10 tokens/sec * 0.1 sec = 1 token)
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Total should be 4 (3 initial + 1 refilled)
	if allowedCount != 4 {
		t.Errorf("Expected 4 total allowed (3 initial + 1 refilled), got %d", allowedCount)
	}
}
