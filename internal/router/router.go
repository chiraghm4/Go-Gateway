package router

import (
	"api-gateway/internal/loadbalancer"
	"api-gateway/internal/middleware"
	"log"
	"net/http"
)

func SetupRoutes(upstreams []string) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	rateLimiter := middleware.NewRateLimiter(250, 250)

	// userLB, err := loadbalancer.NewRoundRobin([]string{
	// 	"http://localhost:8081",
	// 	"http://localhost:8082",
	// 	"http://localhost:8083",
	// })

	userLB, err := loadbalancer.NewRoundRobin(upstreams)
	if err != nil {
		return nil, err
	}

	orderLB, err := loadbalancer.NewRoundRobin(upstreams)
	if err != nil {
		return nil, err
	}

	userLB.StartHealthCheck()
	// orderLB.StartHealthCheck() // no need of this health check

	usersHandler := middleware.Chain(userLB, rateLimiter.Middleware, middleware.LoggingMiddleware)
	ordersHandler := middleware.Chain(orderLB, rateLimiter.Middleware, middleware.LoggingMiddleware)

	mux.Handle("/users", usersHandler)
	mux.Handle("/orders", ordersHandler)

	log.Println("routes are configured")

	return mux, nil
}
