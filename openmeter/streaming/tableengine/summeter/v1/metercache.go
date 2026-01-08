package summeterv1

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

var defaultPeriodicRefetchInterval = 5 * time.Second

type MeterCache struct {
	FilterFn func(meter.Meter) bool

	meterService meter.Service

	mu     sync.RWMutex
	meters map[string]meter.Meter
	logger *slog.Logger

	isRunning atomic.Bool
}

func NewMeterCache(filterFn func(meter.Meter) bool, meterService meter.Service) *MeterCache {
	return &MeterCache{
		FilterFn:     filterFn,
		meterService: meterService,
		meters:       nil,
	}
}

func (c *MeterCache) Start(ctx context.Context) error {
	if c.isRunning.Swap(true) {
		return errors.New("meter cache is already running")
	}

	go c.start(ctx)
	return nil
}

func (c *MeterCache) start(ctx context.Context) {
	periodicRefetchTicker := time.NewTicker(defaultPeriodicRefetchInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-periodicRefetchTicker.C:
			err := c.reloadMeters(ctx)
			if err != nil {
				// TODO: This should be critical after a while
				c.logger.ErrorContext(ctx, "failed to reload meters", "error", err)
			}
		}
	}
}

func (c *MeterCache) reloadMeters(ctx context.Context) error {
	meters, err := c.meterService.ListMeters(ctx, meter.ListMetersParams{
		WithoutNamespace: true,
	})
	if err != nil {
		return err
	}

	updatedMeters := make(map[string]meter.Meter, len(meters.Items))
	for _, meter := range meters.Items {
		if c.FilterFn(meter) {
			updatedMeters[meter.ID] = meter
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.meters = updatedMeters

	return nil
}

func (c *MeterCache) GetMetersByEventTypeNamespace(ctx context.Context, eventType string, namespace string) ([]meter.Meter, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	meters := make([]meter.Meter, 0)
	for _, meter := range c.meters {
		if meter.EventType == eventType && meter.Namespace == namespace {
			meters = append(meters, meter)
		}
	}

	return meters, nil
}
