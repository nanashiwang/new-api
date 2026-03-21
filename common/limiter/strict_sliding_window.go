package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

//go:embed lua/strict_sliding_window.lua
var strictSlidingWindowScript string

var strictSlidingWindowRedisScript = redis.NewScript(strictSlidingWindowScript)

//go:embed lua/query_sliding_window.lua
var querySlidingWindowScript string

var querySlidingWindowRedisScript = redis.NewScript(querySlidingWindowScript)

// QuerySlidingWindowStatus performs a read-only query on the sliding window.
// Returns the current count, oldest entry timestamp (ms), and server now (ms).
func QuerySlidingWindowStatus(
	ctx context.Context,
	client *redis.Client,
	key string,
	window time.Duration,
) (count int64, oldestMs int64, nowMs int64, err error) {
	if client == nil {
		return 0, 0, 0, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return 0, 0, 0, fmt.Errorf("sliding window key is empty")
	}
	if window <= 0 {
		return 0, 0, 0, fmt.Errorf("sliding window duration must be positive")
	}

	result, err := querySlidingWindowRedisScript.Run(
		ctx,
		client,
		[]string{key},
		window.Milliseconds(),
	).Int64Slice()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("query sliding window status failed: %w", err)
	}
	if len(result) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected result length: %d", len(result))
	}
	return result[0], result[1], result[2], nil
}

func AllowStrictSlidingWindow(
	ctx context.Context,
	client *redis.Client,
	key string,
	window time.Duration,
	maxRequests int64,
	nonce string,
	expiration time.Duration,
) (bool, error) {
	if client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	if key == "" {
		return false, fmt.Errorf("sliding window key is empty")
	}
	if window <= 0 {
		return false, fmt.Errorf("sliding window duration must be positive")
	}
	if maxRequests <= 0 {
		return false, fmt.Errorf("sliding window maxRequests must be positive")
	}
	if nonce == "" {
		return false, fmt.Errorf("sliding window nonce is empty")
	}
	if expiration <= 0 {
		expiration = window
	}

	result, err := strictSlidingWindowRedisScript.Run(
		ctx,
		client,
		[]string{key},
		window.Milliseconds(),
		maxRequests,
		nonce,
		expiration.Milliseconds(),
	).Int()
	if err != nil {
		return false, fmt.Errorf("strict sliding window limit failed: %w", err)
	}
	return result == 1, nil
}
