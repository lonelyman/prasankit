package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) Limiter {
	return Limiter{
		redis: redisClient,
	}
}

func (l Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit < 1 {
		return false, fmt.Errorf("rate limit must be greater than 0")
	}
	if window <= 0 {
		return false, fmt.Errorf("rate limit window must be greater than 0")
	}

	count, err := l.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		if err := l.redis.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	}

	return count <= int64(limit), nil
}
