package router

import (
	"api-gateway/internal/proxy"
	"api-gateway/internal/middleware"
	"log"
	"net/http"
)

func SetupRoutes() (*http.ServeMux, error) {

	mux := http.NewServeMux()

	usersProxy, err := proxy.NewReverseProxy("http://localhost:8081")
	if err != nil {
		return nil, err
	}

	ordersProxy, err := proxy.NewReverseProxy("http://localhost:8082")
	if err != nil {
		return nil, err
	}

	usersHandler := middleware.Chain(usersProxy, middleware.LoggingMiddleware)
	ordersHandler := middleware.Chain(ordersProxy, middleware.LoggingMiddleware)

	mux.Handle("/users", usersHandler)
	mux.Handle("/orders", ordersHandler)

	log.Println("routes are configured")

	return mux, nil
}