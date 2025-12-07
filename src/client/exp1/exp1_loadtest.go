package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{Timeout: 5 * time.Second}
	urlFIFO := "http://localhost:8080/task/fifo"
	urlPQ := "http://localhost:8080/task/pq"

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
