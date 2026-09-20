package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// using this script ensures atomicity
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clearBefore = now - window

redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)
local currentRequests = redis.call('ZCARD', key)

if currentRequests < limit then
    redis.call('ZADD', key, now, now .. '-' .. currentRequests)
    redis.call('EXPIRE', key, math.ceil(window / 1000) + 1)
    return {1, limit - currentRequests - 1}
else
    return {0, 0}
end
`

type RateLimiter struct {
	client *RedisClient
	script *redis.Script
}

type RateLimitResult struct {
	Allowed   bool
	Remaining int
	Limit     int
	ResetIn   time.Duration
}

func NewRateLimiter(client *RedisClient) *RateLimiter {
	return &RateLimiter{
		client: client,
		script: redis.NewScript(slidingWindowScript),
	}
}

// Allow checks whether the given key is allowed to make a request within the window.
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()

	redisKey := "ratelimit:" + key
	keys := []string{redisKey}
	args := []any{now, windowMs, limit}

	res, err := rl.script.Run(ctx, rl.client.RDB, keys, args...).Slice()
	if err != nil {
		return nil, err
	}

	allowedInt, _ := res[0].(int64)
	remainingInt, _ := res[1].(int64)

	return &RateLimitResult{
		Allowed:   allowedInt == 1,
		Remaining: int(remainingInt),
		Limit:     limit,
		ResetIn:   window,
	}, nil
}
