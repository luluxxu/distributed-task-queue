package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Task struct {
	ID          string    `json:"id"`
	JobType     string    `json:"job_type"`
	Status      string    `json:"status"`
	RetryCount  int       `json:"retry_count"`
	Error       string    `json:"error"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type CreateTaskResp struct {
	Message string `json:"message"`
	Task    Task   `json:"task"`
}

func main() {
	// baseURL := os.Getenv("API_ENDPOINT")
	// if baseURL == "" {
	// 	baseURL = "http://localhost:8080"
	// }
	baseURL := "http://localhost:8080"
	fmt.Println("Using API endpoint:", baseURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	total := 1000
	taskIDs := make([]string, 0, total)

	var (
		status2xx   int
		status429   int
		status5xx   int
		statusOther int
	)

	fmt.Printf("Submitting %d tasks ...\n", total)
	for i := 0; i < total; i++ {
		reqBody := map[string]string{
			"job_type": "short",
			"payload":  fmt.Sprintf("exp3-data-%d", i),
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			fmt.Println("marshal error:", err)
			continue
		}

		url := baseURL + "/task/pq"
		resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			fmt.Printf("request %d error: %v\n", i, err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status2xx++
		} else if resp.StatusCode == http.StatusTooManyRequests { // 429
			status429++
		} else if resp.StatusCode >= 500 {
			status5xx++
		} else {
			statusOther++
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			continue
		}

		var createResp CreateTaskResp
		if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
			fmt.Printf("request %d decode error: %v\n", i, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		taskIDs = append(taskIDs, createResp.Task.ID)

		if (i+1)%50 == 0 {
			fmt.Printf("  submitted %d/%d tasks\n", i+1, total)
		}
	}

	fmt.Println("=========== HTTP STATUS SUMMARY ===========")
	fmt.Printf("Total requests sent: %d\n", total)
	fmt.Printf("2xx (accepted tasks):     %d\n", status2xx)
	fmt.Printf("429 (rate limited):       %d\n", status429)
	fmt.Printf("5xx (server errors):      %d\n", status5xx)
	fmt.Printf("Other status codes:       %d\n", statusOther)
	fmt.Println("===========================================")

	fmt.Printf("Submitted %d tasks (actually enqueued), waiting for workers to process (and retry)...\n", len(taskIDs))

	wait := 5 * time.Minute
	fmt.Printf("Sleeping %v before fetching task results...\n", wait)
	time.Sleep(wait)

	var (
		successNoRetry   int
		successWithRetry int
		failed           int
		pending          int
		totalRetries     int
	)

	exampleRetried := []*Task{}
	exampleFailed := []*Task{}

	for _, id := range taskIDs {
		task, err := fetchTask(client, baseURL, id)
		if err != nil {
			fmt.Printf("fetch task %s error: %v\n", id, err)
			continue
		}

		totalRetries += task.RetryCount

		switch task.Status {
		case "success":
			if task.RetryCount > 0 {
				successWithRetry++
				if len(exampleRetried) < 5 {
					exampleRetried = append(exampleRetried, task)
				}
			} else {
				successNoRetry++
			}
		case "failed":
			failed++
			if len(exampleFailed) < 5 {
				exampleFailed = append(exampleFailed, task)
			}
		default:
			pending++
		}
	}

	fmt.Println("==================== EXPERIMENT 3 REPORT ====================")
	fmt.Printf("API endpoint: %s\n", baseURL)
	fmt.Printf("Total tasks (enqueued): %d\n", len(taskIDs))
	fmt.Printf("Success (no retry):     %d\n", successNoRetry)
	fmt.Printf("Success (with retry):   %d\n", successWithRetry)
	fmt.Printf("Failed:                 %d\n", failed)
	fmt.Printf("Pending/Running:        %d\n", pending)
	fmt.Printf("Total retries:          %d\n", totalRetries)
	fmt.Println("=============================================================")

	if len(exampleRetried) > 0 {
		fmt.Println("\nExamples: success with retry")
		for _, t := range exampleRetried {
			fmt.Printf("- id=%s status=%s retry_count=%d\n",
				t.ID, t.Status, t.RetryCount)
		}
	}

	if len(exampleFailed) > 0 {
		fmt.Println("\nExamples: failed tasks")
		for _, t := range exampleFailed {
			fmt.Printf("- id=%s status=%s retry_count=%d error=%s\n",
				t.ID, t.Status, t.RetryCount, t.Error)
		}
	} else {
		fmt.Println("\nNo failed tasks observed in this run (all succeeded eventually).")
	}
}

func fetchTask(client *http.Client, baseURL, id string) (*Task, error) {
	url := fmt.Sprintf("%s/task/%s", baseURL, id)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status=%s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}
