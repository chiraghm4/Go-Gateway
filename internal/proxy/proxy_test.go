package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"api-gateway/internal/proxy"
)

func TestNewReverseProxy_ValidTarget(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(backend.Close)

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatalf("NewReverseProxy(%q) returned error: %v", backend.URL, err)
	}
	if p == nil {
		t.Error("NewReverseProxy returned nil")
	}
}

func TestNewReverseProxy_InvalidTarget(t *testing.T) {
	_, err := proxy.NewReverseProxy("://invalid-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestReverseProxy_ForwardsRequests(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello from backend"))
	}))
	t.Cleanup(backend.Close)

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "hello from backend" {
		t.Errorf("body = %q, want %q", got, "hello from backend")
	}
}

func TestReverseProxy_PreservesMethod(t *testing.T) {
	var gotMethod string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
	}))
	t.Cleanup(backend.Close)

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	p.ServeHTTP(httptest.NewRecorder(), req)

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
}

func TestReverseProxy_PreservesHeaders(t *testing.T) {
	var gotHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
	}))
	t.Cleanup(backend.Close)

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "test-value")
	p.ServeHTTP(httptest.NewRecorder(), req)

	if gotHeader != "test-value" {
		t.Errorf("header = %q, want %q", gotHeader, "test-value")
	}
}

func TestReverseProxy_PreservesQueryString(t *testing.T) {
	var gotQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
	}))
	t.Cleanup(backend.Close)

	p, err := proxy.NewReverseProxy(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar&baz=qux", nil)
	p.ServeHTTP(httptest.NewRecorder(), req)

	if gotQuery != "foo=bar&baz=qux" {
		t.Errorf("query = %q, want %q", gotQuery, "foo=bar&baz=qux")
	}
}

func TestReverseProxy_ErrorHandler(t *testing.T) {
	p, err := proxy.NewReverseProxy("http://localhost:59999")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}
