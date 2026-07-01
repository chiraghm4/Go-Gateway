package router

import (
	"api-gateway/internal/loadbalancer"
	"api-gateway/internal/middleware"
	"log"
	"net/http"
)

func SetupRoutes() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	rateLimiter := middleware.NewRateLimiter(250, 250)

	// ordersProxy, err := proxy.NewReverseProxy("http://localhost:8082")
	// if err != nil {
	// 	return nil, err
	// }

	userLB, err := loadbalancer.NewRoundRobin([]string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	})
	
	if err != nil {
		return nil, err
	}

	userLB.StartHealthCheck()

	usersHandler := middleware.Chain(userLB, rateLimiter.Middleware, middleware.LoggingMiddleware)
	// ordersHandler := middleware.Chain(ordersProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)

	mux.Handle("/users", usersHandler)
	// mux.Handle("/orders", ordersHandler)

	log.Println("routes are configured")

	return mux, nil
}
