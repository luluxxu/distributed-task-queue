package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	models "github.com/yourusername/distributed-task-queue/src/api/models"
)

var (
	rdb *redis.Client
	ctx = context.Background()
)

const (
	FIFO_QUEUE_KEY     = "task:queue"
	PRIORITY_QUEUE_KEY = "task:priority_queue"
	TASK_RESULT_PREFIX = "task:result:"
	RETRY_ZSET_KEY     = "task:retry"
	// TASK_TTL is the expiration time for task storage (7 days)
	TASK_TTL = 7 * 24 * time.Hour
)

func InitRedis() {
	// Get Redis address from environment variable, default to localhost
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr, // Use environment variable
		Password: "",
		DB:       0,
	})

	// Test Redis connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	log.Println("✓ Connected to Redis")
}

// CloseRedis closes the Redis client connection
func CloseRedis() {
	if rdb != nil {
		rdb.Close()
		log.Println("✓ Closed Redis connection")
	}
}

// ============================================
// FIFO Queue Operations
// ============================================

// EnqueueFIFO adds a task ID to the FIFO queue
func EnqueueFIFO(taskID string) error {
	return rdb.LPush(ctx, FIFO_QUEUE_KEY, taskID).Err()
}

// DequeueFIFO removes and returns a task ID from the FIFO queue
func DequeueFIFO() (string, error) {
	return rdb.RPop(ctx, FIFO_QUEUE_KEY).Result()
}

// GetFIFOQueueLength returns the number of tasks in the FIFO queue
func GetFIFOQueueLength() (int64, error) {
	return rdb.LLen(ctx, FIFO_QUEUE_KEY).Result()
}

// ClearFIFOQueue removes all tasks from the FIFO queue
func ClearFIFOQueue() error {
	return rdb.Del(ctx, FIFO_QUEUE_KEY).Err()
}

// ============================================
// Priority Queue Operations
// ============================================

// EnqueuePriority adds a task ID to the priority queue
// Short jobs get score 1.0 (higher priority)
// Long jobs get score 2.0 (lower priority)
func EnqueuePriority(taskID string, jobType string) error {
	score := 2.0 // Default: long jobs (lower priority)
	if jobType == "short" {
		score = 1.0 // Short jobs (higher priority)
	}

	return rdb.ZAdd(ctx, PRIORITY_QUEUE_KEY, &redis.Z{
		Score:  score,
		Member: taskID,
	}).Err()
}

// DequeuePriority removes and returns the highest priority task ID
func DequeuePriority() (string, error) {
	result := rdb.ZPopMin(ctx, PRIORITY_QUEUE_KEY, 1).Val()
	if len(result) == 0 {
		return "", redis.Nil
	}
	return result[0].Member.(string), nil
}

// GetPriorityQueueLength returns the number of tasks in the priority queue
func GetPriorityQueueLength() (int64, error) {
	return rdb.ZCard(ctx, PRIORITY_QUEUE_KEY).Result()
}

// ClearPriorityQueue removes all tasks from the priority queue
func ClearPriorityQueue() error {
	return rdb.Del(ctx, PRIORITY_QUEUE_KEY).Err()
}

// ============================================
// Task Storage Operations (Redis STRING)
// ============================================

// StoreTask stores a task in Redis as a JSON string with TTL
func StoreTask(task *models.Task) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	key := TASK_RESULT_PREFIX + task.ID
	return rdb.Set(ctx, key, taskJSON, TASK_TTL).Err()
}

// GetTask retrieves a task from Redis by ID
func GetTask(taskID string) (*models.Task, error) {
	key := TASK_RESULT_PREFIX + taskID
	taskJSON, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var task models.Task
	err = json.Unmarshal([]byte(taskJSON), &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// TaskExists checks if a task exists in Redis (for idempotency)
func TaskExists(taskID string) (bool, error) {
	key := TASK_RESULT_PREFIX + taskID
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// DeleteTask removes a task from Redis
func DeleteTask(taskID string) error {
	key := TASK_RESULT_PREFIX + taskID
	return rdb.Del(ctx, key).Err()
}

// GetAllTaskKeys returns all task keys (for cleanup/analysis)
func GetAllTaskKeys() ([]string, error) {
	return rdb.Keys(ctx, TASK_RESULT_PREFIX+"*").Result()
}

// UpdateTaskStatus updates just the status field of a task
// This is a helper function to avoid retrieving and storing the entire task
func UpdateTaskStatus(taskID string, status string) error {
	task, err := GetTask(taskID)
	if err != nil {
		return err
	}

	task.Status = status
	return StoreTask(task)
}

// ============================================
// Utility Functions
// ============================================

// ClearAllData clears all queues and tasks (useful for testing)
func ClearAllData() error {
	// Clear FIFO queue
	if err := ClearFIFOQueue(); err != nil {
		return err
	}

	// Clear priority queue
	if err := ClearPriorityQueue(); err != nil {
		return err
	}

	// Clear all tasks
	keys, err := GetAllTaskKeys()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return rdb.Del(ctx, keys...).Err()
	}

	return nil
}

// GetRedisClient returns the Redis client (for advanced operations)
func GetRedisClient() *redis.Client {
	return rdb
}

// ============================================
// Retry Queue Operations (for Experiment 3)
// ============================================

// ScheduleRetry adds a task ID to the retry ZSET with the next retry timestamp as score.
func ScheduleRetry(taskID string, next time.Time) error {
	return rdb.ZAdd(ctx, RETRY_ZSET_KEY, &redis.Z{
		Score:  float64(next.Unix()),
		Member: taskID,
	}).Err()
}

// PopDueRetries pops up to 'limit' task IDs whose scheduled retry time is <= now.
// It removes them from the retry ZSET and returns their IDs.
func PopDueRetries(limit int) ([]string, error) {
	now := float64(time.Now().Unix())

	items, err := rdb.ZRangeByScore(ctx, RETRY_ZSET_KEY, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	members := make([]interface{}, len(items))
	for i, it := range items {
		members[i] = it
	}
	if err := rdb.ZRem(ctx, RETRY_ZSET_KEY, members...).Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// ReenqueueByType re-enqueues a task into the appropriate queue based on its JobType.
// For SJF experiment we push back into the priority queue.

func ReenqueueByType(taskID string) error {
	task, err := GetTask(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	switch task.JobType {
	case "short":
		return EnqueuePriority(taskID, "short")
	case "long":
		return EnqueuePriority(taskID, "long")
	default:
		return EnqueueFIFO(taskID)
	}
}
