package experiments

import (
	// "fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	models "github.com/yourusername/distributed-task-queue/src/api/models"
	// rl "github.com/yourusername/distributed-task-queue/src/api/ratelimit"
	redis "github.com/yourusername/distributed-task-queue/src/redis"
)

func Experiment1(router *gin.Engine) {
	// associate GET HTTP method and "/task/:id" path with a handler function "getTaskById"
	router.GET("/task/:id", getTaskByID)
	// associate POST HTTP method and "/task/fifo" path with a handler function "postTaskFIFO"
	router.POST("/task/fifo", postTaskFIFO)
	// associate POST HTTP method and "/task/pq" path with a handler function "postTaskPQ"
	router.POST("/task/pq", postTaskPQ)
}

// postTaskFIFO handles task submission from client to FIFO Queue
func postTaskFIFO(c *gin.Context) {

	// rate limit
	// clientID := c.ClientIP()
	// rateLimitResult, err := rl.Allow(c.Request.Context(), clientID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"error": "rate limiter error",
	// 	})
	// 	return
	// }

	// Set standard rate limit headers for all responses
	//c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitResult.Limit))
	//c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitResult.Remaining))

	// if !rateLimitResult.Allowed {
	// 	// Set Retry-After header when rate limited
	// 	c.Header("Retry-After", fmt.Sprintf("%d", rateLimitResult.RetryAfter))
	// 	c.JSON(http.StatusTooManyRequests, gin.H{
	// 		"error":       "rate limit exceeded",
	// 		"remaining":   rateLimitResult.Remaining,
	// 		"retry_after": rateLimitResult.RetryAfter,
	// 	})
	// 	return
	// }

	var req models.TaskRequest

	// Bind and validate JSON request
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate job_type
	if req.JobType != "short" && req.JobType != "long" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_type must be 'short' or 'long'",
		})
		return
	}

	// Generate task ID if not provided (for idempotency)
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	taskID := req.ID

	// Check for duplicate task ID (idempotency)
	exists, err := redis.TaskExists(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check task existence",
		})
		return
	}

	if exists {
		// Task already exists, return existing task info
		existingTask, err := redis.GetTask(taskID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve existing task",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Task already exists",
			"task":    existingTask,
		})
		return
	}

	// Create new task
	task := models.Task{
		ID:          taskID,
		JobType:     req.JobType,
		Payload:     req.Payload,
		Status:      "queued",
		SubmittedAt: time.Now(),
		RetryCount:  0,
	}

	// Store task in Redis
	err = redis.StoreTask(&task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store task",
		})
		return
	}

	// Enqueue task to FIFO queue
	err = redis.EnqueueFIFO(task.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to enqueue task",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully (FIFO queue)",
		"task":    task,
	})
}

// postTaskPQ handles task submission to priority queue
func postTaskPQ(c *gin.Context) {
	// clientID := c.ClientIP()
	// rateLimitResult, err := rl.Allow(c.Request.Context(), clientID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{
	// 		"error": "rate limiter error",
	// 	})
	// 	return
	// }

	// Set standard rate limit headers for all responses
	// c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitResult.Limit))
	// c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitResult.Remaining))

	// if !rateLimitResult.Allowed {
	// 	// Set Retry-After header when rate limited
	// 	c.Header("Retry-After", fmt.Sprintf("%d", rateLimitResult.RetryAfter))
	// 	c.JSON(http.StatusTooManyRequests, gin.H{
	// 		"error":       "rate limit exceeded",
	// 		"remaining":   rateLimitResult.Remaining,
	// 		"retry_after": rateLimitResult.RetryAfter,
	// 	})
	// 	return
	// }

	var req models.TaskRequest

	// Bind and validate JSON request
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate job_type
	if req.JobType != "short" && req.JobType != "long" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_type must be 'short' or 'long'",
		})
		return
	}

	// Generate task ID if not provided
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	taskID := req.ID

	// Check for duplicate task ID (idempotency)
	exists, err := redis.TaskExists(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check task existence",
		})
		return
	}

	if exists {
		// Task already exists, return existing task info
		existingTask, err := redis.GetTask(taskID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve existing task",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Task already exists",
			"task":    existingTask,
		})
		return
	}

	// Create new task
	task := models.Task{
		ID:          taskID,
		JobType:     req.JobType,
		Payload:     req.Payload,
		Status:      "queued",
		SubmittedAt: time.Now(),
		RetryCount:  0,
	}

	// Store task in Redis
	err = redis.StoreTask(&task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store task",
		})
		return
	}

	// Enqueue task to PRIORITY queue (short jobs get higher priority)
	err = redis.EnqueuePriority(task.ID, task.JobType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to enqueue task",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully (Priority queue)",
		"task":    task,
	})
}

// getTaskByByID locates the task whose ID value matches the id
// parameter sent by the client, then returns the task status as a response.
func getTaskByID(c *gin.Context) {
	taskID := c.Param("id")

	task, err := redis.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Task not found",
			"id":    taskID,
		})
		return
	}

	c.JSON(http.StatusOK, task)
}
