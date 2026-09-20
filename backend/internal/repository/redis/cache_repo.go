package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"url-shortener/internal/models"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type CacheRepository struct {
	client *RedisClient
}

func NewCacheRepository(client *RedisClient) *CacheRepository {
	return &CacheRepository{client: client}
}

func (r *CacheRepository) urlKey(slug string) string {
	return fmt.Sprintf("url:%s", slug)
}

func (r *CacheRepository) GetURL(ctx context.Context, slug string) (*models.CachedURL, error) {
	val, err := r.client.RDB.Get(ctx, r.urlKey(slug)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	var cached models.CachedURL
	if err := json.Unmarshal([]byte(val), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

func (r *CacheRepository) SetURL(ctx context.Context, slug string, cached *models.CachedURL, ttl time.Duration) error {
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	return r.client.RDB.Set(ctx, r.urlKey(slug), data, ttl).Err()
}

func (r *CacheRepository) DeleteURL(ctx context.Context, slug string) error {
	return r.client.RDB.Del(ctx, r.urlKey(slug)).Err()
}
