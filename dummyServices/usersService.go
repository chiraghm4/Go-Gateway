// users.go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Users service hit")
		w.Write([]byte("Users Service Response"))
	})

	fmt.Println("Users service running on :8081")
	http.ListenAndServe(":8081", nil)
}