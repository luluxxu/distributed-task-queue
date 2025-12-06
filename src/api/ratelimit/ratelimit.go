package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/yourusername/distributed-task-queue/src/redis"
)

// maximum number of requests allowed per client per minute
const rateLimitPerMinute = 100

func Allow(ctx context.Context, clientID string) (bool, int, error) {
	// All requests within the same minute will share the same window
	window := time.Now().Format("200601021504") // e.g. 202512051630
	key := fmt.Sprintf("rl:%s:%s", clientID, window)

	rc := redis.GetRedisClient()

	// Atomically increment the counter for this client in the current window
	count, err := rc.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	// Set 2-minute expiration ONLY on first request (count == 1) to auto-clean old keys.
	if count == 1 {
		rc.Expire(ctx, key, 2*time.Minute)
	}
	// Calculate remaining requests in the current window
	remaining := rateLimitPerMinute - int(count)
	if remaining < 0 {
		remaining = 0
	}
	// Determine if the request is allowed
	allowed := count <= int64(rateLimitPerMinute)
	return allowed, remaining, nil
}
