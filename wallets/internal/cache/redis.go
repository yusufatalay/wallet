package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis, creates a connection to redis.
func ConnectRedis(ctx context.Context, addr string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("could not ping redis: %w", err)
	}

	return rdb, nil
}
