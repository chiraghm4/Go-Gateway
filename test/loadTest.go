package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func sendRequest(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get("http://localhost:8080/users")
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}

	fmt.Println("Worker", id, "Status:", resp.StatusCode)

	resp.Body.Close()
}

func main() {

	totalRequests := 1000

	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < totalRequests; i++ {

		wg.Add(1)

		go sendRequest(i, &wg)

	}

	wg.Wait()

	duration := time.Since(start)

	fmt.Println("Completed in:", duration)
}