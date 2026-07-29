package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"api-gateway/internal/middleware"
)

func TestChain_EmptyChain(t *testing.T) {
	var served bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	})

	chained := middleware.Chain(h)
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !served {
		t.Error("chain with no middlewares did not call handler")
	}
}

func TestChain_ExecutionOrder(t *testing.T) {
	var order []int

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, 3)
	})

	// Chain(h, mw2, mw1) → mw2(mw1(handler))
	// Execution: mw2 → mw1 → handler → order [2, 1, 3]
	chained := middleware.Chain(handler, mw2, mw1)
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []int{2, 1, 3}
	if len(order) != len(want) {
		t.Fatalf("execution order = %v, want %v", order, want)
	}
	for i, v := range order {
		if v != want[i] {
			t.Errorf("step %d = %d, want %d", i, v, want[i])
		}
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	var called bool
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	var served bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	})

	chained := middleware.Chain(h, mw)
	chained.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Error("middleware was not called")
	}
	if !served {
		t.Error("handler was not called")
	}
}
