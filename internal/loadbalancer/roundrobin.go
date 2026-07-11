package loadbalancer

import (
	"bytes"
	"io"
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
	alive        atomic.Bool
	failureCount atomic.Uint64
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

		srv := &Server{
			URL:   u,
			Proxy: proxy,
			cb:    circuitbreaker.New(5, 30*time.Second),
		}
		srv.alive.Store(true)
		servers = append(servers, srv)
	}

	return &RoundRobin{
		servers: servers,
	}, nil
}

func (rr *RoundRobin) NextServer() *Server {
	startIndex := int((atomic.AddUint64(&rr.counter, 1) - 1) % uint64(len(rr.servers)))

	for i := 0; i < len(rr.servers); i++ {
		index := (startIndex + i) % len(rr.servers)
		server := rr.servers[index]

		if server.alive.Load() {
			return server
		}
	}

	return nil
}

type captureWriter struct {
	w          http.ResponseWriter
	statusCode int
	written    bool
}

func (cw *captureWriter) Header() http.Header {
	return cw.w.Header()
}

func (cw *captureWriter) Write(b []byte) (int, error) {
	if !cw.written {
		cw.w.WriteHeader(cw.statusCode)
		cw.written = true
	}
	return cw.w.Write(b)
}

func (cw *captureWriter) WriteHeader(code int) {
	if !cw.written {
		cw.statusCode = code
		cw.w.WriteHeader(code)
		cw.written = true
	}
}

func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	maxRetries := len(rr.servers) - 1

	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}

	for i := 0; i <= maxRetries; i++ {
		server := rr.NextServer()
		if server == nil {
			http.Error(w, "no healthy backends", http.StatusBadGateway)
			return
		}

		if !server.cb.Allow() {
			continue
		}

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		cw := &captureWriter{w: w, statusCode: http.StatusOK}
		server.Proxy.ServeHTTP(cw, r)

		if cw.statusCode < 500 {
			server.failureCount.Store(0)
			server.cb.RecordSuccess()
			return
		}

		count := server.failureCount.Add(1)
		if count >= 3 {
			server.alive.Store(false)
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
					server.alive.Store(false)
				} else {
					server.alive.Store(true)
					server.failureCount.Store(0)
				}

				if resp != nil {
					resp.Body.Close()
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

func (rr *RoundRobin) ServerCount() int {
	return len(rr.servers)
}

func (rr *RoundRobin) GetServer(index int) *Server {
	if index < 0 || index >= len(rr.servers) {
		return nil
	}
	return rr.servers[index]
}

func (rr *RoundRobin) SetServerAlive(index int, alive bool) {
	if index >= 0 && index < len(rr.servers) {
		rr.servers[index].alive.Store(alive)
	}
}

func (rr *RoundRobin) IsServerAlive(index int) bool {
	if index >= 0 && index < len(rr.servers) {
		return rr.servers[index].alive.Load()
	}
	return false
}

func (rr *RoundRobin) CircuitBreakerState(index int) circuitbreaker.State {
	if index >= 0 && index < len(rr.servers) {
		return rr.servers[index].cb.State()
	}
	return circuitbreaker.StateClosed
}

func (rr *RoundRobin) ResetFailureCount(index int) {
	if index >= 0 && index < len(rr.servers) {
		rr.servers[index].failureCount.Store(0)
	}
}
