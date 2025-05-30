package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	redisClient *redis.Client
}

func NewRedisCache(redisClient *redis.Client) (Cache, error) {
	return &redisCache{
		redisClient: redisClient,
	}, nil
}

func (c *redisCache) SetHighWatermark(ctx context.Context, key NamespacedKey, highWatermark HighWatermarkCacheEntry) error {
	keyStr := key.GetNamespacedKey()
	deletedKeyStr := fmt.Sprintf("%s:del", keyStr)

	// let's check if the key is already deleted
	_, err := c.redisClient.Get(ctx, deletedKeyStr).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	if err == nil {
		// The deleted key exists, so we don't want to set a high watermark.
		return nil
	}

	if highWatermark.IsDeleted {
		// let's set the deleted key
		return c.redisClient.SetArgs(ctx, deletedKeyStr, "", redis.SetArgs{
			TTL: defaultHighWatermarkCacheTTL,
		}).Err()
	}

	highWatermarkTS := highWatermark.HighWatermark.UnixNano()
	// TODO: exponential backoff
	for {
		// We need to set the high watermark key, let's set up optimistic locking
		err = c.redisClient.Watch(ctx, func(tx *redis.Tx) error {
			existingValue, err := tx.Get(ctx, keyStr).Result()
			if err != nil {
				if !errors.Is(err, redis.Nil) {
					return err
				}
			}

			existingValueInt, err := strconv.ParseInt(existingValue, 10, 64)
			if err != nil {
				// High watermark key failed to parse => treat as non-existing
				existingValueInt = 0
			}

			if existingValueInt >= highWatermarkTS {
				return nil
			}

			return tx.SetArgs(ctx, keyStr, highWatermarkTS, redis.SetArgs{
				TTL: defaultHighWatermarkCacheTTL,
			}).Err()
		}, keyStr)
		if err == nil {
			break
		}

		// The error is not due to high watermark key was modified by another process
		if !errors.Is(err, redis.TxFailedErr) {
			return err
		}

		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

func (c *redisCache) GetHighWatermark(ctx context.Context, key NamespacedKey) (HighWatermarkCacheEntry, error) {
	keyStr := key.GetNamespacedKey()
	deletedKeyStr := fmt.Sprintf("%s:del", keyStr)

	// let's check if the key is already deleted
	results, err := c.redisClient.MGet(ctx, keyStr, deletedKeyStr).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return HighWatermarkCacheEntry{}, err
	}

	if errors.Is(err, redis.Nil) || len(results) == 0 {
		return HighWatermarkCacheEntry{}, nil
	}

	highWatermark := HighWatermarkCacheEntry{
		HighWatermark: time.Unix(0, results[0].(int64)).In(time.UTC), // In(time.UTC) forces normalization of the time
		IsDeleted:     results[1] != nil,
	}

	return highWatermark, nil
}
