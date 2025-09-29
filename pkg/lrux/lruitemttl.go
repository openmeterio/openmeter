package lrux

import (
	"context"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/pkg/clock"
)

// CacheWithItemTTL is a cache that fetches items with an optional TTL
// if the TTL has been reached the item is fetched again, otherwise the item is returned from the cache
type CacheWithItemTTL[K comparable, V any] struct {
	*lru.Cache[K, CacheItemWithTTL[V]]

	fetcher Fetcher[K, V]
	ttl     time.Duration
}

type CacheItemWithTTL[V any] struct {
	Value     V
	ExpiresAt time.Time
}

type Fetcher[K comparable, V any] func(context.Context, K) (V, error)

type cacheOptions struct {
	ttl time.Duration
}

type cacheOptionsFunc func(*cacheOptions)

func WithTTL(ttl time.Duration) cacheOptionsFunc {
	return func(o *cacheOptions) {
		o.ttl = ttl
	}
}

func NewCacheWithItemTTL[K comparable, V any](size int, fetcher Fetcher[K, V], opts ...cacheOptionsFunc) (*CacheWithItemTTL[K, V], error) {
	cacheOptions := cacheOptions{}

	for _, opt := range opts {
		opt(&cacheOptions)
	}

	if fetcher == nil {
		return nil, fmt.Errorf("fetcher is required")
	}

	if cacheOptions.ttl < 0 {
		return nil, fmt.Errorf("ttl must be positive or 0")
	}

	cache, err := lru.New[K, CacheItemWithTTL[V]](size)
	if err != nil {
		return nil, err
	}

	return &CacheWithItemTTL[K, V]{
		Cache:   cache,
		fetcher: fetcher,
		ttl:     cacheOptions.ttl,
	}, nil
}

// Get fetches the item from the cache if it exists and is not expired, otherwise it fetches the item from the fetcher
func (c *CacheWithItemTTL[K, V]) Get(ctx context.Context, key K) (V, error) {
	item, ok := c.Cache.Get(key)
	if ok && (item.ExpiresAt.IsZero() || item.ExpiresAt.After(clock.Now())) {
		return item.Value, nil
	}

	// We need to fetch the item
	// NOTE: we are not using a mutex here, as we don't want to limit the number of fetches for now

	item, err := c.fetchItem(ctx, key)
	if err != nil {
		var empty V
		return empty, err
	}

	c.Cache.Add(key, item)

	return item.Value, nil
}

// Refresh fetches the item from the fetcher and updates the cache
func (c *CacheWithItemTTL[K, V]) Refresh(ctx context.Context, key K) (V, error) {
	item, err := c.fetchItem(ctx, key)
	if err != nil {
		var empty V
		return empty, err
	}

	c.Cache.Add(key, item)

	return item.Value, nil
}

func (c *CacheWithItemTTL[K, V]) fetchItem(ctx context.Context, key K) (CacheItemWithTTL[V], error) {
	item, err := c.fetcher(ctx, key)
	if err != nil {
		return CacheItemWithTTL[V]{}, err
	}

	return CacheItemWithTTL[V]{
		Value:     item,
		ExpiresAt: clock.Now().Add(c.ttl),
	}, nil
}
