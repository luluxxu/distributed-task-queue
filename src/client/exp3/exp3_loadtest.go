package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TaskRequest struct {
	JobType string `json:"job_type"`
	Payload string `json:"payload"`
}

func main() {

	baseURL := "http://localhost:8080"
	url := baseURL + "/task/fifo"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	total := 500

	for i := 0; i < total; i++ {
		reqBody := TaskRequest{
			JobType: "short",
			Payload: fmt.Sprintf("exp3-data-%d", i),
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Println("marshal error:", err)
			continue
		}

		resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("request %d error: %v\n", i, err)
			continue
		}

		fmt.Printf("Submitted #%d to %s, status=%s\n", i, url, resp.Status)
		resp.Body.Close()

	}

	fmt.Println("Experiment 3 loadtest finished.")
}
