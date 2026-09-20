package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	RDB *redis.Client
}

func (c *RedisClient) Close() error {
	if c.RDB != nil {
		c.RDB.Close()
	}
	return nil
}

func NewRedisClient(ctx context.Context, addr, password string, db int) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis at %s: %w", addr, err)
	}

	log.Println("Connected to Redis successfully")
	return &RedisClient{RDB: rdb}, nil
}
