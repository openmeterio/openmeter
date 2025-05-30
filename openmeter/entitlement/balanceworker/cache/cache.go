package cache

import (
	"context"
	"time"
)

const (
	defaultHighWatermarkCacheTTL = 24 * time.Hour
)

type HighWatermarkCacheEntry struct {
	HighWatermark time.Time
	IsDeleted     bool
}

type NamespacedKey interface {
	GetNamespacedKey() string
}

type Cache interface {
	SetHighWatermark(ctx context.Context, key NamespacedKey, highWatermark HighWatermarkCacheEntry) error
	GetHighWatermark(ctx context.Context, key NamespacedKey) (HighWatermarkCacheEntry, error)
}
