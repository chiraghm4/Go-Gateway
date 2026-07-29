package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	go us1()
	go us2()
	go us3() 

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig 
	fmt.Println("shutting down...")
}

func us1() {
	mux := http.NewServeMux()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-1 called")
		w.Write([]byte("users-service-1"))
	})

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("orders-service-1 called")
		w.Write([]byte("orders-service-1"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("health check passed")
		w.Write([]byte("health check passed - us1"))
	})

	fmt.Println("running users-service-1 on 8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		fmt.Printf("us1 server error: %v\n", err)
	}
}

func us2() {
	mux := http.NewServeMux()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-2 called")
		w.Write([]byte("users-service-2"))
	})

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("orders-service-2 called")
		w.Write([]byte("orders-service-2"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("health check passed")
		w.Write([]byte("health check passed - us2"))
	})

	fmt.Println("running users-service-2 on 8082")
	
	if err := http.ListenAndServe(":8082", mux); err != nil {
		fmt.Printf("us2 server error: %v\n", err)
	}
}

func us3() {
	mux := http.NewServeMux()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-3 called")
		w.Write([]byte("users-service-3"))
	})

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("orders-service-3 called")
		w.Write([]byte("orders-service-3"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("health check passed")
		w.Write([]byte("health check passed - us3"))
	})

	fmt.Println("running users-service-3 on 8083")
	if err := http.ListenAndServe(":8083", mux); err != nil {
		fmt.Printf("us3 server error: %v\n", err)
	}
}