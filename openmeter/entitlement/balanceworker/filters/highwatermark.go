package filters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
)

const (
	// defaultClockDrift specifies how much clock drift is allowed when calculating the current time between the worker nodes.
	// with AWS, Google Cloud 1ms is guaranteed, this should work well for any NTP based setup.
	defaultClockDrift = time.Millisecond
)

var (
	_ NamedFilter             = (*HighWatermarkCache)(nil)
	_ CalculationTimeRecorder = (*HighWatermarkCache)(nil)
)

type highWatermarkCacheEntry struct {
	HighWatermark time.Time `json:"wm"`
	IsDeleted     bool      `json:"del"`
}

type HighWatermarkCache struct {
	backend HighWatermarkBackend
}

func NewHighWatermarkCache(backend HighWatermarkBackend) (*HighWatermarkCache, error) {
	if backend == nil {
		return nil, errors.New("backend is required")
	}

	return &HighWatermarkCache{backend: backend}, nil
}

func (f *HighWatermarkCache) Name() string {
	return "highwatermark"
}

func (f *HighWatermarkCache) IsNamespaceInScope(ctx context.Context, namespace string) (bool, error) {
	return true, nil
}

func (f *HighWatermarkCache) IsEntitlementInScope(ctx context.Context, req EntitlementFilterRequest) (bool, error) {
	if err := req.Validate(); err != nil {
		return false, err
	}

	if req.Operation == snapshot.ValueOperationReset {
		// Reset events are always in scope, so that notification has a fresh event for the new period
		return true, nil
	}

	entry, err := f.backend.Get(ctx, req.Entitlement.ID)
	if err != nil {
		return false, err
	}

	if !entry.IsPresent {
		// Not found in cache => we consider it to be in scope
		return true, nil
	}

	if entry.IsDeleted {
		// Deleted entitlement => we consider it to be out of scope regardless of the high watermark
		return false, nil
	}

	if req.EventAt.After(entry.HighWatermark.Add(-defaultClockDrift)) {
		// Event is after the high watermark => we consider it to be in scope
		return true, nil
	}

	// Event is before the high watermark => we consider it to be out of scope
	return false, nil
}

func (f *HighWatermarkCache) RecordLastCalculation(ctx context.Context, req RecordLastCalculationRequest) error {
	return f.backend.Record(ctx, req)
}

// Backend interface

type highWatermarkBackendGetResult struct {
	highWatermarkCacheEntry
	IsPresent bool
}

type HighWatermarkBackend interface {
	Get(ctx context.Context, entitlementID string) (highWatermarkBackendGetResult, error)
	Record(ctx context.Context, req RecordLastCalculationRequest) error
}

// In memory backend

var _ HighWatermarkBackend = (*HighWatermarkInMemoryBackend)(nil)

type HighWatermarkInMemoryBackend struct {
	cache *lru.Cache[string, highWatermarkCacheEntry]
}

func NewHighWatermarkInMemoryBackend(cacheSize int) (*HighWatermarkInMemoryBackend, error) {
	cache, err := lru.New[string, highWatermarkCacheEntry](cacheSize)
	if err != nil {
		return nil, err
	}

	return &HighWatermarkInMemoryBackend{cache: cache}, nil
}

func (b *HighWatermarkInMemoryBackend) Get(ctx context.Context, entitlementID string) (highWatermarkBackendGetResult, error) {
	entry, ok := b.cache.Get(entitlementID)
	if !ok {
		return highWatermarkBackendGetResult{IsPresent: false}, nil
	}

	return highWatermarkBackendGetResult{IsPresent: true, highWatermarkCacheEntry: entry}, nil
}

func (b *HighWatermarkInMemoryBackend) Record(ctx context.Context, req RecordLastCalculationRequest) error {
	b.cache.Add(req.Entitlement.ID, highWatermarkCacheEntry{
		HighWatermark: req.CalculatedAt,
		IsDeleted:     req.IsDeleted,
	})

	return nil
}

// Redis backend

var _ HighWatermarkBackend = (*HighWatermarkRedisBackend)(nil)

type HighWatermarkRedisBackendConfig struct {
	Redis      *redis.Client
	Logger     *slog.Logger
	Expiration time.Duration
}

func (c HighWatermarkRedisBackendConfig) Validate() error {
	if c.Redis == nil {
		return errors.New("redis is required")
	}

	if c.Expiration <= 0 {
		return errors.New("ttl must be greater than 0")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

type HighWatermarkRedisBackend struct {
	HighWatermarkRedisBackendConfig
}

func NewHighWatermarkRedisBackend(cfg HighWatermarkRedisBackendConfig) (*HighWatermarkRedisBackend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &HighWatermarkRedisBackend{
		HighWatermarkRedisBackendConfig: cfg,
	}, nil
}

func (b *HighWatermarkRedisBackend) getCacheKey(entitlementID string) string {
	return fmt.Sprintf("hw.v1:%s", entitlementID)
}

func (b *HighWatermarkRedisBackend) Get(ctx context.Context, entitlementID string) (highWatermarkBackendGetResult, error) {
	cacheKey := b.getCacheKey(entitlementID)

	return b.getEntry(ctx, b.Redis, cacheKey)
}

func (b *HighWatermarkRedisBackend) getEntry(ctx context.Context, tx redis.Cmdable, cacheKey string) (highWatermarkBackendGetResult, error) {
	highWatermark, err := tx.Get(ctx, cacheKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return highWatermarkBackendGetResult{IsPresent: false}, nil
		}

		return highWatermarkBackendGetResult{IsPresent: false}, err
	}

	entry := highWatermarkCacheEntry{}
	if err := json.Unmarshal([]byte(highWatermark), &entry); err != nil {
		return highWatermarkBackendGetResult{IsPresent: false}, err
	}

	return highWatermarkBackendGetResult{IsPresent: true, highWatermarkCacheEntry: entry}, nil
}

func (b *HighWatermarkRedisBackend) Record(ctx context.Context, req RecordLastCalculationRequest) error {
	cacheKey := b.getCacheKey(req.Entitlement.ID)

	// Let's start an optimistic lock on the cache key
	err := b.Redis.Watch(ctx, func(tx *redis.Tx) error {
		entry, err := b.getEntry(ctx, tx, cacheKey)
		if err != nil {
			return err
		}

		if !b.shouldUpdateCacheEntry(entry, req) {
			return nil
		}

		newEntry := highWatermarkCacheEntry{
			HighWatermark: req.CalculatedAt,
			IsDeleted:     req.IsDeleted,
		}

		data, err := json.Marshal(newEntry)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			return pipe.SetEx(ctx, cacheKey, string(data), b.Expiration).Err()
		})

		return err
	}, cacheKey)

	// Redis uses optimistic locking, so in case there are multiple updates happening in parallel, one will succeed
	// others will fail with a TxFailedErr. (and could retry)
	//
	// We are just ignoring the TxFailedErr as in such cases we have a recent highwatermark cache entry either ways
	if errors.Is(err, redis.TxFailedErr) {
		b.Logger.Info("high watermark cache update skipped due to parallel updates", "entry.key", cacheKey, "entry.highwatermark", req.CalculatedAt, "entry.isdeleted", req.IsDeleted)
		return nil
	}

	return err
}

func (b *HighWatermarkRedisBackend) shouldUpdateCacheEntry(existing highWatermarkBackendGetResult, req RecordLastCalculationRequest) bool {
	// Entry is missing => we should add it to redis
	if !existing.IsPresent {
		return true
	}

	// Entitlement deletion status changed => we should update the cache
	if req.IsDeleted != existing.IsDeleted {
		return true
	}

	// The entitlement is deleted in the cache => we should just keep it that way
	if existing.IsDeleted {
		return false
	}

	// The new watermark is after the existing watermark => we should update the cache
	if req.CalculatedAt.After(existing.HighWatermark) {
		return true
	}

	return false
}
