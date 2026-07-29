package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-gateway/internal/middleware"
)

func TestLoggingMiddleware_CallsNextHandler(t *testing.T) {
	var served bool
	h := middleware.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !served {
		t.Error("logging middleware did not call next handler")
	}
}

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(nil) })

	h := middleware.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), "GET /test-path") {
		t.Errorf("log output = %q, want it to contain %q", buf.String(), "GET /test-path")
	}
}
