package filters

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

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

func NewHighWatermarkCache(lruCacheSize int) (*HighWatermarkCache, error) {
	backend, err := NewHighWatermarkInMemoryBackend(lruCacheSize)
	if err != nil {
		return nil, err
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
