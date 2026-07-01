package test

import (
	"api-gateway/internal/proxy"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewReverseProxy_ValidTarget(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	p, err := proxy.NewReverseProxy(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to create reverse proxy: %v", err)
	}

	if p == nil {
		t.Error("Expected non-nil proxy")
	}
}

func TestNewReverseProxy_InvalidTarget(t *testing.T) {
	_, err := proxy.NewReverseProxy("://invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestReverseProxy_ForwardsRequests(t *testing.T) {
	// Create backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/test" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello from backend"))
		}
	}))
	defer backend.Close()

	// Create proxy
	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Create test mux and handle request
	mux := http.NewServeMux()
	mux.Handle("/", p)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "Hello from backend" {
		t.Errorf("Expected 'Hello from backend', got '%s'", w.Body.String())
	}
}

func TestReverseProxy_PreservesMethod(t *testing.T) {
	var receivedMethod string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", p)

	// Test POST method
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if receivedMethod != "POST" {
		t.Errorf("Expected POST method, got %s", receivedMethod)
	}
}

func TestReverseProxy_PreservesHeaders(t *testing.T) {
	var receivedHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom-Header")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", p)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if receivedHeader != "test-value" {
		t.Errorf("Expected header 'test-value', got '%s'", receivedHeader)
	}
}

func TestReverseProxy_ErrorHandler(t *testing.T) {
	// Create proxy to non-existent server
	p, err := proxy.NewReverseProxy("http://localhost:59999")
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", p)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should return 502 Bad Gateway
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", w.Code)
	}
}

func TestReverseProxy_PassesQueryString(t *testing.T) {
	var receivedQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", p)

	req := httptest.NewRequest("GET", "/?foo=bar&baz=qux", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if receivedQuery != "foo=bar&baz=qux" {
		t.Errorf("Expected query 'foo=bar&baz=qux', got '%s'", receivedQuery)
	}
}

func TestSingleHostReverseProxy_Director(t *testing.T) {
	// Test that the proxy correctly modifies the request URL
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Verify proxy is properly configured
	if p == nil {
		t.Error("Expected non-nil proxy")
	}
}
