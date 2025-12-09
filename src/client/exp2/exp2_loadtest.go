package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	totalTasks = 500
)

type QueueStatus struct {
	FIFO         int64 `json:"fifo_queue_length"`
	Priority     int64 `json:"priority_queue_length"`
	TotalBacklog int64 `json:"total_backlog"`
}

func main() {
	// Get API endpoint from environment variable
	baseURL := os.Getenv("API_ENDPOINT")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	fmt.Println("Using API endpoint:", baseURL)

	// Only submit to FIFO queue (not priority queue)
	taskSubmitURL := baseURL + "/task/fifo"
	queueStatusURL := baseURL + "/queue/status"

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Printf("Experiment 2: Submitting %d tasks to FIFO queue...\n", totalTasks)

	// Submit all tasks
	for i := 0; i < totalTasks; i++ {
		body, _ := json.Marshal(map[string]string{
			"job_type": "short",
			"payload":  fmt.Sprintf("exp2-%d", i),
		})
		client.Post(taskSubmitURL, "application/json", bytes.NewReader(body))
	}
	submitTime := time.Now()

	// Wait for backlog to clear
	fmt.Println("Waiting for backlog to clear...")
	checkCount := 0
	for {
		backlog := getBacklog(client, queueStatusURL)
		checkCount++

		if backlog == -1 {
			fmt.Printf("Error: Failed to get queue status (check #%d)\n", checkCount)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if backlog == 0 {
			fmt.Printf("✓ Backlog cleared! (checked %d times)\n", checkCount)
			break
		}

		// Print progress every 10 checks (every 5 seconds)
		if checkCount%10 == 0 {
			fmt.Printf("  [Check #%d] Backlog: %d tasks remaining...\n", checkCount, backlog)
		}

		time.Sleep(500 * time.Millisecond)
	}
	clearTime := time.Now()

	// Results
	clearanceTime := clearTime.Sub(submitTime)
	throughput := float64(totalTasks) / clearanceTime.Seconds()

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Tasks: %d\n", totalTasks)
	fmt.Printf("  Clearance time: %v\n", clearanceTime)
	fmt.Printf("  Throughput: %.2f tasks/sec\n", throughput)
}

func getBacklog(client *http.Client, queueStatusURL string) int64 {
	resp, err := client.Get(queueStatusURL)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	var status QueueStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return -1
	}

	// exp 2： FIFO backlog
	return status.FIFO
}
