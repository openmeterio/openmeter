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
	_ NamedFilter             = (*HighWatermarkCacheInMemory)(nil)
	_ CalculationTimeRecorder = (*HighWatermarkCacheInMemory)(nil)
)

type highWatermarkCacheEntry struct {
	HighWatermark time.Time
	IsDeleted     bool
}

type HighWatermarkCacheInMemory struct {
	cache *lru.Cache[string, highWatermarkCacheEntry]
}

func NewHighWatermarkCacheInMemory(cacheSize int) (*HighWatermarkCacheInMemory, error) {
	cache, err := lru.New[string, highWatermarkCacheEntry](cacheSize)
	if err != nil {
		return nil, err
	}

	return &HighWatermarkCacheInMemory{cache: cache}, nil
}

func (f *HighWatermarkCacheInMemory) Name() string {
	return "highwatermark-inmem"
}

func (f *HighWatermarkCacheInMemory) IsNamespaceInScope(ctx context.Context, namespace string) (bool, error) {
	return true, nil
}

func (f *HighWatermarkCacheInMemory) IsEntitlementInScope(ctx context.Context, req EntitlementFilterRequest) (bool, error) {
	if err := req.Validate(); err != nil {
		return false, err
	}

	if req.Operation == snapshot.ValueOperationReset {
		// Reset events are always in scope, so that notification has a fresh event for the new period
		return true, nil
	}

	entry, ok := f.cache.Get(req.Entitlement.ID)
	if !ok {
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

func (f *HighWatermarkCacheInMemory) RecordLastCalculation(ctx context.Context, req RecordLastCalculationRequest) error {
	f.cache.Add(req.Entitlement.ID, highWatermarkCacheEntry{
		HighWatermark: req.CalculatedAt,
		IsDeleted:     req.IsDeleted,
	})

	return nil
}
