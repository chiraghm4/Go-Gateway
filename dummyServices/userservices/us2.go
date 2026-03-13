package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-2 called")
		w.Write([]byte("users-service-2"))
	})

	fmt.Println("running users-service-2 on 8082")
	http.ListenAndServe(":8082", nil)
}