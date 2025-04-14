package estimator

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/go-redsync/redsync/v4"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/samber/lo"
)

type EntitlementCached struct {
	LastCalculation snapshot.EntitlementValue `json:"lastCalculation"`
	LastCalculated  time.Time                 `json:"lastCalculated"`

	// ApproxUsage is the approximate usage of the entitlement, it is guaranteed, that the usage is at least as much as the entitlement balance as now.
	ApproxUsage InfDecimal `json:"approxUsage"`
}

type IngestEventInput struct {
	Target TargetEntitlement

	DedupedEvents []serializer.CloudEventsKafkaPayload
}

type TargetEntitlement struct {
	Entitlement entitlement.Entitlement
	Feature     feature.Feature
	Meter       meter.Meter
}

func (e TargetEntitlement) Validate() error {
	if e.Entitlement.ID == "" {
		return errors.New("entitlement ID is required")
	}

	if e.Feature.ID == "" {
		return errors.New("feature ID is required")
	}

	if e.Meter.ID == "" {
		return errors.New("meter ID is required")
	}

	return nil
}

func (t *TargetEntitlement) GetEntryHash() hasher.Hash {
	// TODO!: nil safety + validation! (e.g CurrentUsagePeriod)

	// TODO: how to express granting with a hash
	// Entitlement entity changes
	val := strings.Join(
		[]string{
			t.Entitlement.ID, t.Entitlement.UpdatedAt.String(), lo.FromPtr(t.Entitlement.DeletedAt).String(),
			// TODO: NR of voided and active grants !!!!
			// Entitlement usage period change
			t.Entitlement.CurrentUsagePeriod.From.String(), t.Entitlement.CurrentUsagePeriod.To.String(),
			// Feature changes
			t.Feature.ID, t.Feature.UpdatedAt.String(), lo.FromPtr(t.Feature.ArchivedAt).String(),
			// Meter changes
			t.Meter.ID, t.Meter.UpdatedAt.String(), lo.FromPtr(t.Meter.DeletedAt).String(),
		}, ":",
	)
	return hasher.NewHash([]byte(val))
}

type HandleRecalculationResult struct {
	Value            *snapshot.EntitlementValue
	CalculationError error
}

type Cache interface {
	HandleEntitlementEvent(ctx context.Context, event IngestEventInput) (EntitlementCached, error)
	HandleRecalculation(ctx context.Context, target TargetEntitlement, calculationFn func(ctx context.Context) (*snapshot.EntitlementValue, error)) (*snapshot.EntitlementValue, error)
	Remove(ctx context.Context, target TargetEntitlement) error
	IntrospectCacheEntry(ctx context.Context, target TargetEntitlement) (*EntitlementCached, error)
}

type CacheOptions struct {
	RedisURL    string
	LockTimeout time.Duration
	Logger      *slog.Logger
}

func (o *CacheOptions) Validate() error {
	if o.RedisURL == "" {
		return errors.New("redisURL is required")
	}

	if o.LockTimeout <= 0 {
		return errors.New("lockTimeout must be greater than 0")
	}

	if o.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func NewCache(in CacheOptions) (Cache, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	backend, err := NewRedisCacheBackend(RedisCacheBackendOptions{
		RedisURL:    in.RedisURL,
		LockTimeout: in.LockTimeout,
		Logger:      in.Logger,
	})
	if err != nil {
		return nil, err
	}

	return &cache{
		backend: backend,
		logger:  in.Logger,
	}, nil
}

type cache struct {
	backend CacheBackend

	logger *slog.Logger
}

var _ Cache = (*cache)(nil)

var (
	ErrUnsupportedMeterAggregation = errors.New("unsupported meter aggregation")
	ErrEntryNotFound               = errors.New("entry not found")
	ErrLockAlreadyExpired          = redsync.ErrLockAlreadyExpired
)

// TODO: test getChange
func (c *cache) HandleEntitlementEvent(ctx context.Context, event IngestEventInput) (EntitlementCached, error) {
	entry, err := c.backend.UpdateCacheEntryIfExists(ctx, event.Target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		// TODO: return error
		if entry == nil {
			return nil, ErrEntryNotFound
		}

		var approxIncrease InfDecimal
		switch event.Target.Meter.Aggregation {
		// Meters that we can calculate exactly
		case meter.MeterAggregationSum:
			// TODO: calculate
			getChange, err := c.getChange(event.Target.Meter, event.DedupedEvents)
			if err != nil {
				// TODO: more details
				c.logger.Warn("failed to get change for entitlement", "error", err)
				// We should not fail the whole calculation, instead if we have any pending
				// thresholds, this will force a recalculation of the entitlement balance.
				approxIncrease = infinite
				break
			}

			if getChange.IsNegative() {
				// Negative change is not possible as we might apply the event twice =>
				// let's just ignore it, this way we maintain the invariant that the usage is always >= entitlement balance
				break
			}

			approxIncrease = getChange
		case meter.MeterAggregationCount:
			// TODO: this is an approximation, as we ignore the group by filters, but it satisfies
			// that cached usage is always >= entitlement balance
			approxIncrease = NewInfDecimal(float64(len(event.DedupedEvents)))
		// Meters that we cannot calculate but we can have approximations for
		case meter.MeterAggregationUniqueCount:
			approxIncrease = NewInfDecimal(float64(len(event.DedupedEvents)))
		default:
			// Avg, Min, Max
			return nil, ErrUnsupportedMeterAggregation
		}

		entry.ApproxUsage = entry.ApproxUsage.Add(approxIncrease)
		return entry, nil
	})
	if err != nil {
		return EntitlementCached{}, err
	}

	if entry == nil {
		// TODO: maybe seperate error?!
		return EntitlementCached{}, ErrEntryNotFound
	}

	return *entry, nil
}

func (c *cache) getChange(meterDef meter.Meter, events []serializer.CloudEventsKafkaPayload) (InfDecimal, error) {
	totalEventValue := alpacadecimal.Zero

	for _, rawEvent := range events {
		parsedEvent, err := meter.ParseEvent(meterDef, []byte(rawEvent.Data))
		if err != nil {
			// TODO: Details
			return InfDecimal{}, err
		}

		if parsedEvent.Value != nil {
			if *parsedEvent.Value > 0 {
				totalEventValue = totalEventValue.Add(alpacadecimal.NewFromFloat(*parsedEvent.Value))
			}
		} else {
			return infinite, nil
		}
	}

	return NewInfDecimalFromDecimal(totalEventValue), nil
}

func (c *cache) Remove(ctx context.Context, target TargetEntitlement) error {
	return c.backend.Delete(ctx, target)
}

func (c *cache) HandleRecalculation(ctx context.Context, target TargetEntitlement, calculationFn func(ctx context.Context) (*snapshot.EntitlementValue, error)) (*snapshot.EntitlementValue, error) {
	cacheEntry, err := c.backend.UpdateCacheEntryIfExists(ctx, target, func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error) {
		res, err := calculationFn(ctx)
		if err != nil {
			return nil, err
		}

		return &EntitlementCached{
			LastCalculation: *res,
			LastCalculated:  time.Now(),
			ApproxUsage:     NewInfDecimal(lo.FromPtr(res.Usage)),
		}, nil
	})
	if err != nil {
		if errors.Is(err, ErrLockAlreadyExpired) {
			if cacheEntry != nil {
				return &cacheEntry.LastCalculation, nil
			}
		}
		return nil, err
	}

	return &cacheEntry.LastCalculation, nil
}

func (c *cache) IntrospectCacheEntry(ctx context.Context, target TargetEntitlement) (*EntitlementCached, error) {
	return c.backend.Get(ctx, target)
}
