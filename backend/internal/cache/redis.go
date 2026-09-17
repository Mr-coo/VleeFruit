// Package cache wires up the Redis client.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedis creates a Redis client and verifies it with a ping.
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
