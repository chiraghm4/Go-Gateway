package router

import (
	"api-gateway/internal/loadbalancer"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"log"
	"net/http"
)

func SetupRoutes() (*http.ServeMux, error) {

	mux := http.NewServeMux()

	// NewRateLimiter(capacity int, refillRate int)
	rateLimiter := middleware.NewRateLimiter(250, 250)

	// usersProxy, err := proxy.NewReverseProxy("http://localhost:8081")
	// if err != nil {
	// 	return nil, err
	// }

	ordersProxy, err := proxy.NewReverseProxy("http://localhost:8082")
	if err != nil {
		return nil, err
	}

	userLB, err := loadbalancer.NewRoundRobin([]string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	})
	if err != nil {
		return nil, err
	}

	userLB.StartHealthCheck()

	// urlShortnerHandler := middleware.Chain(urlShortnerProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)
	usersHandler := middleware.Chain(userLB, rateLimiter.Middleware, middleware.LoggingMiddleware)
	ordersHandler := middleware.Chain(ordersProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)

	// mux.Handle("/", urlShortnerHandler)
	// mux.Handle("/shorten", urlShortnerHandler)
	mux.Handle("/users", usersHandler)
	mux.Handle("/orders", ordersHandler)

	log.Println("routes are configured")	

	return mux, nil
}
