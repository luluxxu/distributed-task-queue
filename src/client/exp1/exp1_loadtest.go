package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	// get API endpoint from environment variable
	baseURL := os.Getenv("API_ENDPOINT")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	fmt.Println("Using API endpoint:", baseURL)
	// build URLs
	urlFIFO := baseURL + "/task/fifo"
	urlPQ := baseURL + "/task/pq"

	client := &http.Client{Timeout: 5 * time.Second}

	total := 200 // total tasks
	for i := 0; i < total; i++ {
		jobType := "short"
		if i%5 == 0 { // 20% long jobs, 80% short jobs
			jobType = "long"
		}

		body, _ := json.Marshal(map[string]string{
			"job_type": jobType,
			"payload":  fmt.Sprintf("exp1-data-%d", i),
		})

		// half FIFO，half PQ
		var url string
		if i%2 == 0 {
			url = urlFIFO
		} else {
			url = urlPQ
		}

		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println("Submitted", i, "to", url, "status:", resp.Status)
		resp.Body.Close()
	}
}
