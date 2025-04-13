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
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

type EntitlementCached struct {
	Target TargetEntitlement

	LastCalculation snapshot.EntitlementValue
	LastCalculated  time.Time

	// ApproxUsage is the approximate usage of the entitlement, it is guaranteed, that the usage is at least as much as the entitlement balance as now.
	ApproxUsage InfDecimal
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

type Cache interface {
	HandleEntitlementEvent(ctx context.Context, event IngestEventInput) (*EntitlementCached, error)
	HandleRecalculation(ctx context.Context, target TargetEntitlement, calculationFn func(ctx context.Context) (*snapshot.EntitlementValue, error)) (*snapshot.EntitlementValue, error)
	Remove(ctx context.Context, target TargetEntitlement) error
}

type cache struct {
	data    map[hasher.Hash]EntitlementCached
	backend CacheBackend
}

var ErrUnsupportedMeterAggregation = errors.New("unsupported meter aggregation")

func (c *cache) HandleEntitlementEvent(ctx context.Context, event IngestEventInput) (EntitlementCached, error) {
	entry, err := c.backend.UpdateCacheEntryIfExists(ctx, event.Target, func(ctx context.Context, entry EntitlementCached) (EntitlementCached, error) {
		hash := event.Target.GetEntryHash()

		// Something have happend with the entitlement or it's dependencies, we need to re-evaluate the entitlement
		if entry.ConsistencyHash != hash {
			return EntitlementCached{}, ErrConcurrentUpdate
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
			return entry, ErrUnsupportedMeterAggregation
		}

		entry.ApproxUsage = entry.ApproxUsage.Add(approxIncrease)
		return entry, nil
	})
	if err != nil {
		return EntitlementCached{}, err
	}

	return entry, nil
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

var (
	// Cache entry was updated concurrently
	ErrConcurrentUpdate = errors.New("concurrent update")
	ErrEntryNotFound    = errors.New("entry not found")
)

type CacheBackend interface {
	UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry EntitlementCached) (EntitlementCached, error)) (EntitlementCached, error)
}

type redisCacheBackend struct {
	redis *redis.Client
}

const (
	cacheEntryStatusPending = iota
	cacheEntryStatusValid
)

type wrappedCacheEntry struct {
	status int
	entry  EntitlementCached
}

func (b *redisCacheBackend) UpdateCacheEntryIfExists(ctx context.Context, target TargetEntitlement, updater func(ctx context.Context, entry *EntitlementCached) (EntitlementCached, error)) (EntitlementCached, error) {
	var cachedEntry EntitlementCached

	key := fmt.Sprintf("entitlement:%s.%x", target.Entitlement.ID, target.GetEntryHash())

	wrappedEntryUpsert := wrappedCacheEntry{
		status: cacheEntryStatusPending,
		entry:  cachedEntry,
	}

	wrappedEntryUpsertJSON, err := json.Marshal(wrappedEntryUpsert)
	if err != nil {
		return EntitlementCached{}, err
	}

	// Let's upsert the entry into the cache
	upserted, err := b.redis.SetNX(ctx, key, wrappedEntryUpsertJSON, time.Hour).Result()
	if err != nil {
		return EntitlementCached{}, err
	}

	// TODO:
	if !upserted {
		return EntitlementCached{}, ErrConcurrentUpdate
	}

	err = b.redis.Watch(ctx, func(tx *redis.Tx) error {
		entryJSON, err := tx.Get(ctx, key).Result()
		if err != nil {
			return err
		}

		currentEntry := wrappedCacheEntry{}
		err = json.Unmarshal([]byte(entryJSON), &currentEntry)
		if err != nil {
			return err
		}

		var cacheEntryIn *EntitlementCached
		if currentEntry.status == cacheEntryStatusValid {
			cacheEntryIn = &currentEntry.entry
		}

		newEntry, err := updater(ctx, cacheEntryIn)
		if err != nil {
			return err
		}

		_, err = tx.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			newEntryJSON, err := json.Marshal(newEntry)
			if err != nil {
				return err
			}

			pipe.Set(ctx, key, newEntryJSON, time.Hour)
			return nil
		})
		if err != nil {
			return err
		}

		cachedEntry = newEntry
		return nil
	})
	if err != nil {
		// TODO: map redis errors to our own error types
		return EntitlementCached{}, err
	}

	return cachedEntry, nil
}
