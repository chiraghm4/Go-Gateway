package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Orders service hit")
		w.Write([]byte("Orders Service Response"))
	})

	fmt.Println("Orders service running on :8082")
	http.ListenAndServe(":8082", nil)
}