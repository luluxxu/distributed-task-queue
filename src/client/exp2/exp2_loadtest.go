package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	totalTasks = 1000
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

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("Experiment 2: Submitting %d tasks to FIFO queue...\n", totalTasks)

	// Submit all tasks concurrently for faster submission
	submitTime := time.Now()
	var wg sync.WaitGroup
	concurrency := 100                      // Increase concurrency to create backlog (100 concurrent requests)
	sem := make(chan struct{}, concurrency) // Semaphore to limit concurrent requests

	for i := 0; i < totalTasks; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			body, _ := json.Marshal(map[string]string{
				"job_type": "short",
				"payload":  fmt.Sprintf("exp2-%d", taskNum),
			})
			resp, err := client.Post(taskSubmitURL, "application/json", bytes.NewReader(body))
			if err != nil {
				fmt.Printf("Error submitting task %d: %v\n", taskNum, err)
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	submitEndTime := time.Now()
	fmt.Printf("All %d tasks submitted in %v\n", totalTasks, submitEndTime.Sub(submitTime))

	// Wait for backlog to clear and track peak backlog
	fmt.Println("Waiting for backlog to clear...")
	checkCount := 0
	maxBacklog := int64(0)
	processingStartTime := submitEndTime // Start of processing phase

	for {
		backlog := getBacklog(client, queueStatusURL)
		checkCount++

		if backlog == -1 {
			// Only print error every 10 checks to avoid spam
			if checkCount%10 == 0 {
				fmt.Printf("Warning: Failed to get queue status (check #%d) - retrying...\n", checkCount)
			}
			time.Sleep(1 * time.Second) // Wait longer on error
			continue
		}

		// Track peak backlog
		if backlog > maxBacklog {
			maxBacklog = backlog
		}

		if backlog == 0 {
			fmt.Printf("✓ Backlog cleared! (checked %d times)\n", checkCount)
			break
		}

		// Print progress every 10 checks (every 5 seconds)
		if checkCount%10 == 0 {
			fmt.Printf("  [Check #%d] Backlog: %d tasks remaining... (peak: %d)\n", checkCount, backlog, maxBacklog)
		}

		time.Sleep(500 * time.Millisecond)
	}
	clearTime := time.Now()

	// Results
	submitDuration := submitEndTime.Sub(submitTime)
	processingDuration := clearTime.Sub(processingStartTime)
	totalDuration := clearTime.Sub(submitTime)
	throughput := float64(totalTasks) / totalDuration.Seconds()
	processingThroughput := float64(totalTasks) / processingDuration.Seconds()

	separator := strings.Repeat("=", 70)
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("EXPERIMENT 2 RESULTS\n")
	fmt.Printf("%s\n", separator)
	fmt.Printf("Total Tasks:           %d\n", totalTasks)
	fmt.Printf("Submission Time:       %v (%.2f tasks/sec)\n", submitDuration, float64(totalTasks)/submitDuration.Seconds())
	fmt.Printf("Processing Time:       %v (%.2f tasks/sec)\n", processingDuration, processingThroughput)
	fmt.Printf("Total Clearance Time:  %v\n", totalDuration)
	fmt.Printf("Overall Throughput:    %.2f tasks/sec\n", throughput)
	fmt.Printf("Peak Queue Backlog:    %d tasks\n", maxBacklog)
	fmt.Printf("%s\n", separator)

	// CSV format for easy recording
	fmt.Printf("\nCSV Format (for recording):\n")
	fmt.Printf("tasks,submit_time_sec,processing_time_sec,total_time_sec,throughput_tasks_per_sec,peak_backlog\n")
	fmt.Printf("%d,%.2f,%.2f,%.2f,%.2f,%d\n",
		totalTasks,
		submitDuration.Seconds(),
		processingDuration.Seconds(),
		totalDuration.Seconds(),
		throughput,
		maxBacklog)
}

func getBacklog(client *http.Client, queueStatusURL string) int64 {
	resp, err := client.Get(queueStatusURL)
	if err != nil {
		// Network error or timeout
		return -1
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// API returned an error (e.g., 500 Internal Server Error)
		return -1
	}

	var status QueueStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		// JSON decode error
		return -1
	}

	// exp 2： FIFO backlog
	return status.FIFO
}
