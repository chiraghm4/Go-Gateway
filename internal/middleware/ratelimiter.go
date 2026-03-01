package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// token bucket implementation

type visitor struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	visitors   map[string]*visitor
	mu         sync.Mutex
	capacity   float64
	refillRate float64 // tokens per second
}

func NewRateLimiter(capacity int, refillRate int) *RateLimiter {
	return &RateLimiter{
		visitors:   make(map[string]*visitor),
		capacity:   float64(capacity),
		refillRate: float64(refillRate),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		rl.mu.Lock()
		defer rl.mu.Unlock()
		
		v, exists := rl.visitors[ip]

		if !exists {
			rl.visitors[ip] = &visitor{
				tokens: rl.capacity - 1,
				lastRefill: time.Now(),
			}
			next.ServeHTTP(w, r)
			return
		}
		
		// refill tokens
		now := time.Now()
		elapsed := now.Sub(v.lastRefill).Seconds()

		v.tokens += elapsed * rl.refillRate

		if v.tokens > rl.capacity {
			v.tokens = rl.capacity
		}

		v.lastRefill = now

		// check if token available
		if v.tokens < 1 {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		// consume token
		v.tokens--

		next.ServeHTTP(w, r)
	})

}
