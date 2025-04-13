package negcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

const (
	// TODO: Parameterize this
	defaultLockExpiry = 4 * time.Second

	defaultSetCacheKeyTimeout = 1 * time.Second
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
			// TODO: NR of voided and active grants
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
}

type cache struct {
	backend CacheBackend
}

var _ Cache = (*cache)(nil)

var (
	ErrUnsupportedMeterAggregation = errors.New("unsupported meter aggregation")
	ErrEntryNotFound               = errors.New("entry not found")
	ErrLockAlreadyExpired          = redsync.ErrLockAlreadyExpired
)

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
				slog.Warn("failed to get change for entitlement", "error", err)
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

type CacheBackend interface {
	UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error)) (*EntitlementCached, error)
	Delete(ctx context.Context, target TargetEntitlement) error
}

type redisCacheBackend struct {
	redis *redis.Client

	redsync *redsync.Redsync
}

func NewRedisCacheBackend(redisURL string) (CacheBackend, error) {
	redisURLParsed, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	redis := redis.NewClient(redisURLParsed)

	return &redisCacheBackend{
		redis:   redis,
		redsync: redsync.New(goredis.NewPool(redis)),
	}, nil
}

type cacheEntryStatus string

const (
	cacheEntryStatusPending cacheEntryStatus = "pending"
	cacheEntryStatusValid   cacheEntryStatus = "valid"
)

type wrappedCacheEntry struct {
	Status cacheEntryStatus `json:"status"`
	// To ensure that values are not equal, we use a random ULID
	RandomULID string             `json:"random"`
	Entry      *EntitlementCached `json:"entry"`
}

func (b *redisCacheBackend) withRedisLock(ctx context.Context, target TargetEntitlement, fn func(ctx context.Context, lock *redsync.Mutex) error) error {
	lockKey := fmt.Sprintf("lock:entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())

	lock := b.redsync.NewMutex(lockKey, redsync.WithExpiry(defaultLockExpiry))

	if err := lock.LockContext(ctx); err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			lock.Unlock()
			panic(recovered)
		}
	}()

	fnErr := fn(ctx, lock)

	// Might return ErrLockAlreadyExpired
	if _, err := lock.UnlockContext(ctx); err != nil {
		// Function might have returned an error, in this case we want to return that error instead of joining
		// the errors (so that the caller can handle the specific callback error and not the lock error)
		if fnErr != nil {
			return fnErr
		}

		return err

	}

	return fnErr
}

func (b *redisCacheBackend) UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry *EntitlementCached) (*EntitlementCached, error)) (*EntitlementCached, error) {
	var newEntry *EntitlementCached

	err := b.withRedisLock(ctx, target, func(ctx context.Context, lock *redsync.Mutex) error {
		key := fmt.Sprintf("entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())

		// TODO: add cache entry ttl as now the entry is always updated thus ttls resets
		// Let's delete the entry in case the lock expires, that way subsequent calls will assume cold cache
		entryJSON, err := b.redis.GetDel(ctx, key).Result()

		notCached := false
		if err != nil {
			if errors.Is(err, redis.Nil) {
				notCached = true
			} else {
				return err
			}
		}

		var existingEntry *EntitlementCached
		if !notCached {
			unmarshaled := EntitlementCached{}
			err = json.Unmarshal([]byte(entryJSON), &unmarshaled)
			if err != nil {
				return err
			}

			existingEntry = &unmarshaled
		}

		newEntry, err = updater(ctx, existingEntry)
		if err != nil {
			return err
		}

		// If the updater returns nil, we do not update the cache
		if newEntry == nil {
			return nil
		}

		newValue, err := json.Marshal(newEntry)
		if err != nil {
			return err
		}

		// We should only update the cache if the lock is not yet expired.
		// It is fine to do the update() call even if the lock is expired, as this might be a recalculation
		// that we would want to execute regardless. This is why we are creating the timeout context here.
		//
		// Let's create a child context with a timeout for the update.

		lockCtx, lockCancel := context.WithDeadline(ctx, lock.Until())
		defer lockCancel()

		_, err = b.redis.Set(lockCtx, key, newValue, time.Hour).Result()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return redsync.ErrLockAlreadyExpired
			}

			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntry, nil
}

func (b *redisCacheBackend) Delete(ctx context.Context, target TargetEntitlement) error {
	return b.withRedisLock(ctx, target, func(ctx context.Context, lock *redsync.Mutex) error {
		key := fmt.Sprintf("entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())
		return b.redis.Del(ctx, key).Err()
	})
}
