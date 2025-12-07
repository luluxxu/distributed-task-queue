package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/yourusername/distributed-task-queue/src/redis"
)

// maximum number of requests allowed per client per minute
const rateLimitPerMinute = 10000

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter int // seconds until window resets
	Limit      int
}

// Allow checks if a request is allowed and returns rate limit information
func Allow(ctx context.Context, clientID string) (*RateLimitResult, error) {
	now := time.Now()
	// All requests within the same minute will share the same window
	window := now.Format("200601021504") // e.g. 202512051630
	key := fmt.Sprintf("rl:%s:%s", clientID, window)

	rc := redis.GetRedisClient()

	// Atomically increment the counter for this client in the current window
	count, err := rc.Incr(ctx, key).Result()
	if err != nil {
		return nil, err
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

	// Calculate retry_after: seconds until the next minute window starts
	// Current window format: 200601021504 (year, month, day, hour, minute)
	// Parse current window to get the minute
	windowTime, err := time.Parse("200601021504", window)
	if err != nil {
		// Fallback: if parsing fails, use 60 seconds
		return &RateLimitResult{
			Allowed:    allowed,
			Remaining:  remaining,
			RetryAfter: 60,
			Limit:      rateLimitPerMinute,
		}, nil
	}

	// Next window starts at the next minute
	nextWindow := windowTime.Add(1 * time.Minute)
	retryAfter := int(nextWindow.Sub(now).Seconds())
	if retryAfter < 0 {
		retryAfter = 0
	}
	if retryAfter > 60 {
		retryAfter = 60 // Cap at 60 seconds
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		Limit:      rateLimitPerMinute,
	}, nil
}
