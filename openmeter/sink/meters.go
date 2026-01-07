package sink

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type MetersByType map[string][]*meter.Meter

type NamespacedMeterCache struct {
	mu        sync.RWMutex
	isRunning atomic.Bool

	namespaces              map[string]MetersByType
	logger                  *slog.Logger
	periodicRefetchInterval time.Duration
	meterService            meter.Service
}

type NamespacedMeterCacheConfig struct {
	PeriodicRefetchInterval time.Duration
	Logger                  *slog.Logger
	MeterService            meter.Service
}

func (c NamespacedMeterCacheConfig) Validate() error {
	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.MeterService == nil {
		return errors.New("meter service is required")
	}

	if c.PeriodicRefetchInterval <= 0 {
		return errors.New("periodic refetch interval must be greater than 0")
	}

	return nil
}

func NewNamespaceStore(config NamespacedMeterCacheConfig) (*NamespacedMeterCache, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &NamespacedMeterCache{
		namespaces:              map[string]MetersByType{},
		meterService:            config.MeterService,
		logger:                  config.Logger,
		periodicRefetchInterval: config.PeriodicRefetchInterval,
	}, nil
}

func (n *NamespacedMeterCache) Start(ctx context.Context) error {
	if n.isRunning.Swap(true) {
		return errors.New("namespaced meter cache is already running")
	}

	meters, err := n.fetchMeters(ctx)
	if err != nil {
		return err
	}

	n.updateCache(meters)
	go n.start(ctx)

	return nil
}

func (n *NamespacedMeterCache) start(ctx context.Context) {
	lastFetch := time.Now()

	periodicRefetchTicker := time.NewTicker(n.periodicRefetchInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-periodicRefetchTicker.C:
			updatedCache, err := n.fetchMeters(ctx)
			if err != nil {
				n.logger.ErrorContext(ctx, "failed to fetch meters", "error", err)
				continue
			}

			n.updateCache(updatedCache)

			n.logger.DebugContext(ctx, "refetched meters", "namespaces", len(n.namespaces), "age", time.Since(lastFetch).String())
			lastFetch = time.Now()
		}
	}
}

func (n *NamespacedMeterCache) fetchMeters(ctx context.Context) (map[string]MetersByType, error) {
	meters, err := n.meterService.ListMeters(ctx, meter.ListMetersParams{
		WithoutNamespace: true,
	})
	if err != nil {
		return nil, err
	}

	metersByType := make(map[string]MetersByType)
	for _, meterEntity := range meters.Items {
		if _, found := metersByType[meterEntity.Namespace]; !found {
			metersByType[meterEntity.Namespace] = MetersByType{}
		}

		if _, found := metersByType[meterEntity.Namespace]; !found {
			metersByType[meterEntity.Namespace] = MetersByType{}
		}

		metersByType[meterEntity.Namespace][meterEntity.EventType] = append(metersByType[meterEntity.Namespace][meterEntity.EventType], lo.ToPtr(meterEntity))
	}

	return metersByType, nil
}

func (n *NamespacedMeterCache) updateCache(meters map[string]MetersByType) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.namespaces = meters
}

// GetAffectedMeters gets the list of meters that are affected by an event
func (n *NamespacedMeterCache) GetAffectedMeters(ctx context.Context, m *sinkmodels.SinkMessage) ([]*meter.Meter, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	namespaceStore, ok := n.namespaces[m.Namespace]
	if !ok || namespaceStore == nil {
		n.logger.WarnContext(ctx, "no namespace store found for namespace", "namespace", m.Namespace)
		return nil, nil
	}

	// We are not interested in processing events that were dropped
	if m.Status.DropError != nil {
		return nil, nil
	}

	return n.namespaces[m.Namespace][m.Serialized.Type], nil
}
