package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	visitors map[string]int
	mu       sync.Mutex
	limit 	 int
}

// Reset map every second
func (rl *RateLimiter) resetCounts() {
	for {
		time.Sleep(1 * time.Second)

		rl.mu.Lock()
		rl.visitors = make(map[string]int)
		rl.mu.Unlock()
	}
}

func NewRateLimiter(limit int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]int),
		limit: limit,
	}

	go rl.resetCounts()
	return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		 ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		 rl.mu.Lock()

		 rl.visitors[ip]++

		 if rl.visitors[ip] > rl.limit {
			rl.mu.Unlock()
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		 }

		 rl.mu.Unlock()

		 next.ServeHTTP(w, r)
	})

}