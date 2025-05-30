package cache

import (
	"context"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

type inProcessCache struct {
	highWatermarkCacheMutex sync.RWMutex
	highWatermarkCache      *lru.Cache[string, HighWatermarkCacheEntry]
}

func NewInProcessCache(size int) (Cache, error) {
	hwCache, err := lru.New[string, HighWatermarkCacheEntry](size)
	if err != nil {
		return nil, err
	}

	return &inProcessCache{
		highWatermarkCache: hwCache,
	}, nil
}

func (c *inProcessCache) SetHighWatermark(ctx context.Context, key NamespacedKey, highWatermark HighWatermarkCacheEntry) error {
	// We need to lock the cache as it does not support CompareAndSwap schemantics
	c.highWatermarkCacheMutex.Lock()
	defer c.highWatermarkCacheMutex.Unlock()

	keyStr := key.GetNamespacedKey()

	prevHighWatermark, prevFound, _ := c.highWatermarkCache.PeekOrAdd(keyStr, highWatermark)

	// We are done, the new item has been added to the cache
	if !prevFound {
		return nil
	}

	if prevHighWatermark.IsDeleted {
		return nil
	}

	// New cache entry is deleted -> let's update the cache
	if highWatermark.IsDeleted {
		_ = c.highWatermarkCache.Add(keyStr, highWatermark)
		return nil
	}

	// We are not decreasing the high watermark, if the cache has a newer entry, we are done
	if prevHighWatermark.HighWatermark.After(highWatermark.HighWatermark) {
		return nil
	}

	_ = c.highWatermarkCache.Add(keyStr, highWatermark)

	return nil
}

func (c *inProcessCache) GetHighWatermark(ctx context.Context, key NamespacedKey) (HighWatermarkCacheEntry, error) {
	c.highWatermarkCacheMutex.RLock()
	defer c.highWatermarkCacheMutex.RUnlock()

	highWatermark, ok := c.highWatermarkCache.Get(key.GetNamespacedKey())
	if !ok {
		return HighWatermarkCacheEntry{}, nil
	}

	return highWatermark, nil
}
