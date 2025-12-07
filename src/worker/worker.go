package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	models "github.com/yourusername/distributed-task-queue/src/api/models"
	r "github.com/yourusername/distributed-task-queue/src/redis"
)

// Global random number generator for simulating failures
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	transientRate = 0.20 // 20% chance of transient (temporary) failure
	permanentRate = 0.05 // 5% chance of permanent failure
	maxRetries    = 5
	baseBackoff   = 200 * time.Millisecond
)

func main() {
	// queueType := "fifo"
	// queueType := "priority"
	queueType := flag.String("queue", "fifo", "Queue type: fifo or priority")
	mode := flag.String("mode", "simple", "simple or retry")
	flag.Parse()

	r.InitRedis()
	defer r.CloseRedis()

	StartWorkerWithQueue(*queueType, *mode)
}

// StartWorkerWithQueue starts a worker with specified queue type
// queueType can be "fifo" or "priority"
func StartWorkerWithQueue(queueType, mode string) {
	log.Printf("Worker started (queue: %s), polling for tasks...", queueType)

	//Start a background goroutine to handle retry scheduling
	if mode == "retry" {
		startRetryScheduler()
	}

	// Infinite loop: continuously poll for tasks
	for {
		var taskID string
		var err error

		// Dequeue from specified queue type
		if queueType == "priority" {
			taskID, err = r.DequeuePriority()
		} else {
			taskID, err = r.DequeueFIFO()
		}

		// Handle empty queue
		if err == redis.Nil {
			// Queue is empty, wait before retrying
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Handle other errors
		if err != nil {
			log.Printf("Error dequeuing task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Get task details from Redis
		task, err := r.GetTask(taskID)
		if err != nil {
			log.Printf("Failed to get task %s: %v", taskID, err)
			continue
		}

		if mode == "retry" {
			processTaskWithFailureAndRetry(task)
		} else {
			processTaskSimple(task)
		}
	}
}

// processTask without retry
func processTaskSimple(task *models.Task) {
	log.Printf("Processing task: %s (type: %s)", task.ID, task.JobType)

	// Update status to running
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	err := r.StoreTask(task)
	if err != nil {
		log.Printf("Failed to update task status to running: %v", err)
		return
	}

	// Simulate work based on job type
	if task.JobType == "short" {
		time.Sleep(50 * time.Millisecond) // Short job: 50ms
	} else if task.JobType == "long" {
		time.Sleep(3 * time.Second) // Long job: 3 seconds
	} else {
		// Unknown job type, default to short
		time.Sleep(50 * time.Millisecond)
	}

	// Update status to success
	completed := time.Now()
	task.Status = "success"
	task.CompletedAt = &completed
	task.Result = "Task completed successfully"
	err = r.StoreTask(task)
	if err != nil {
		log.Printf("Failed to update task status to success: %v", err)
		return
	}

	// Calculate and log latency
	latency := completed.Sub(task.SubmittedAt)
	log.Printf("Completed %s task %s (latency: %v)", task.JobType, task.ID, latency)
}

// processTask with retry
func processTaskWithFailureAndRetry(task *models.Task) {
	log.Printf("Processing task: %s (type: %s)", task.ID, task.JobType)

	// Update status to running
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	err := r.StoreTask(task)
	if err != nil {
		log.Printf("Failed to update task status to running: %v", err)
		return
	}

	// Simulate work based on job type
	if task.JobType == "short" {
		time.Sleep(50 * time.Millisecond) // Short job: 50ms
	} else if task.JobType == "long" {
		time.Sleep(3 * time.Second) // Long job: 3 seconds
	} else {
		// Unknown job type, default to short
		time.Sleep(50 * time.Millisecond)
	}
	// 20% chance of transient failure (can be retried)
	u := rng.Float64() // random float number
	if u < permanentRate {
		finalizeFailed(task, "permanent error")
		return
	} else if u < permanentRate+transientRate { //  0.05 ≤ u < 0.25
		handleTransient(task)
		return
	}

	// Update status to success
	completed := time.Now()
	task.Status = "success"
	task.CompletedAt = &completed
	task.Result = "Task completed successfully"
	err = r.StoreTask(task)
	if err != nil {
		log.Printf("Failed to update task status to success: %v", err)
		return
	}

	// Calculate and log latency
	latency := completed.Sub(task.SubmittedAt)
	log.Printf("Completed %s task %s (latency: %v)", task.JobType, task.ID, latency)
}

// For tasks that are ready to be retried.
// Tasks whose retry time has arrived, and re-enqueues them back into the appropriate queue.
func startRetryScheduler() {
	go func() {
		// Create a ticker that fires every 200ms
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		// Infinite loop: check for retries every 200ms
		for range ticker.C {
			// Query Redis for up to 128 tasks whose retry time has arrived
			ids, err := r.PopDueRetries(128)
			if err != nil {
				log.Printf("retry scan error: %v", err)
				continue
			}
			// Re-enqueue each task back into the appropriate queue
			for _, id := range ids {
				if err := r.ReenqueueByType(id); err != nil {
					log.Printf("reenqueue retry %s error: %v", id, err)
				} else {
					log.Printf("→ Re-enqueued retry task %s", id)
				}
			}
		}
	}()
}

// Mark a task as permanently failed (no more retries)
func finalizeFailed(task *models.Task, reason string) {
	t := time.Now()
	task.Status = "failed"
	task.CompletedAt = &t
	task.Error = reason
	// Save final state to Redis
	if err := r.StoreTask(task); err != nil {
		log.Printf("Failed to store failed task: %v", err)
	}
	log.Printf("Failed task %s (type=%s, reason=%s, retry_count=%d)",
		task.ID, task.JobType, reason, task.RetryCount)
}

func handleTransient(task *models.Task) {
	task.RetryCount++
	// Check if we've exhausted all retry attempts
	if task.RetryCount > maxRetries {
		finalizeFailed(task, "exhausted retries")
		return
	}

	backoff := baseBackoff * time.Duration(1<<uint(task.RetryCount-1))

	// This prevents all failed tasks from retrying at the exact same time
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))

	// Calculate next retry time
	next := time.Now().Add(backoff + jitter)

	// Update task retry count in Redis
	if err := r.StoreTask(task); err != nil {
		log.Printf("Failed to store transient fail attempt: %v", err)
	}
	// Schedule the retry in Redis ZSET
	if err := r.ScheduleRetry(task.ID, next); err != nil {
		log.Printf("Failed to schedule retry for task %s: %v", task.ID, err)
		// Fallback: immediately re-enqueue to avoid losing the task
		_ = r.ReenqueueByType(task.ID)
	}

	log.Printf("Transient failure for task %s (type=%s, retry=%d, next_retry_at=%s)",
		task.ID, task.JobType, task.RetryCount, next.Format(time.RFC3339))
}
