package router

import (
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"log"
	"net/http"
)

func SetupRoutes() (*http.ServeMux, error) {

	mux := http.NewServeMux()

	rateLimiter := middleware.NewRateLimiter(1000, 1000)

	// urlShortnerProxy, err := proxy.NewReverseProxy("http://localhost:8081")
	// if err != nil {
	// 	return nil, err
	// }

	usersProxy, err := proxy.NewReverseProxy("http://localhost:8081")
	if err != nil {
		return nil, err
	}

	ordersProxy, err := proxy.NewReverseProxy("http://localhost:8082")
	if err != nil {
		return nil, err
	}

	// urlShortnerHandler := middleware.Chain(urlShortnerProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)
	usersHandler := middleware.Chain(usersProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)
	ordersHandler := middleware.Chain(ordersProxy, rateLimiter.Middleware, middleware.LoggingMiddleware)

	// mux.Handle("/", urlShortnerHandler)
	// mux.Handle("/shorten", urlShortnerHandler)
	mux.Handle("/users", usersHandler)
	mux.Handle("/orders", ordersHandler)

	log.Println("routes are configured")

	return mux, nil
}
