package main

import (
	"log"
	"flag"
	"time"
	"github.com/go-redis/redis/v8"
	r "github.com/yourusername/distributed-task-queue/src/redis" 
	models "github.com/yourusername/distributed-task-queue/src/api/models"
)

func main() {
	// queueType := "fifo"
	// queueType := "priority"
	queueType := flag.String("queue", "fifo", "Queue type: fifo or priority")
	flag.Parse()
	
	r.InitRedis()
	defer r.CloseRedis()

	StartWorkerWithQueue(*queueType)
}


// StartWorker starts a worker that processes tasks from FIFO queue
func StartWorker() {
	StartWorkerWithQueue("fifo")
}

// StartWorkerWithQueue starts a worker with specified queue type
// queueType can be "fifo" or "priority"
func StartWorkerWithQueue(queueType string) {
	log.Printf("Worker started (queue: %s), polling for tasks...", queueType)

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

		// Process the task
		processTask(task)
	}
}

// processTask executes a task and updates its status
func processTask(task *models.Task) {
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
	log.Printf("✓ Completed %s task %s (latency: %v)", task.JobType, task.ID, latency)
}

// ProcessTaskWithFailure is for Experiment 3 (with failure injection)
// Not needed for Experiment 1, but included for completeness
func ProcessTaskWithFailure(task *models.Task, failureRate float64) {
	log.Printf("Processing task: %s (type: %s)", task.ID, task.JobType)

	// Update status to running
	now := time.Now()
	task.Status = "running"
	task.StartedAt = &now
	r.StoreTask(task)

	// Simulate work
	if task.JobType == "short" {
		time.Sleep(50 * time.Millisecond)
	} else {
		time.Sleep(3 * time.Second)
	}

	// Simulate failure (for Experiment 3)
	// For Experiment 1, we don't use this - all tasks succeed
	// if rand.Float64() < failureRate {
	//     task.Status = "failed"
	//     task.Error = "simulated failure"
	// } else {
	//     task.Status = "success"
	//     task.Result = "Task completed successfully"
	// }

	// For now, always succeed
	completed := time.Now()
	task.Status = "success"
	task.CompletedAt = &completed
	task.Result = "Task completed successfully"
	r.StoreTask(task)

	latency := completed.Sub(task.SubmittedAt)
	log.Printf("✓ Completed %s task %s (latency: %v)", task.JobType, task.ID, latency)
}