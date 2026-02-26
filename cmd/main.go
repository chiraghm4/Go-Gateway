package main

import (
	"log"
	"api-gateway/internal/router"
	"net/http"
	"time"
)

func main() {
	mux, err := router.SetupRoutes()
	if err != nil {
		log.Fatal("Server failed: ", err)
	}

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Println("API Gateway running on :8080")

	log.Fatal(server.ListenAndServe())

}