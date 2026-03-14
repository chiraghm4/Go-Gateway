package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("users-service-3 called")
		w.Write([]byte("users-service-3"))
	})

	fmt.Println("running users-service-3 on 8083")
	http.ListenAndServe(":8083", nil)
}