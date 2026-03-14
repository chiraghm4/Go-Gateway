package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-1 called")
		w.Write([]byte("users-service-1"))
	})

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Println("welcome to user server 1")
	// 	w.Write([]byte("welcome from user server 1"))
	// })

	fmt.Println("running users-service-1 on 8081")
	http.ListenAndServe(":8081", nil)
}