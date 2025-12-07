package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL        = "http://localhost:8080"
	taskSubmitURL  = baseURL + "/task/fifo"
	queueStatusURL = baseURL + "/queue/status"
	totalTasks     = 500
)

type QueueStatus struct {
	TotalBacklog int64 `json:"total_backlog"`
}

func main() {
	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Printf("Experiment 2: Submitting %d tasks...\n", totalTasks)

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
	for {
		backlog := getBacklog(client)
		if backlog == 0 {
			break
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

func getBacklog(client *http.Client) int64 {
	resp, err := client.Get(queueStatusURL)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	var status QueueStatus
	json.NewDecoder(resp.Body).Decode(&status)
	return status.TotalBacklog
}
