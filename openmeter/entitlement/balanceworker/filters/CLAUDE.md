# filters

<!-- archie:ai-start -->

> Pluggable filter chain that gates whether a balance recalculation event should be processed for a given namespace/entitlement. Prevents redundant recalculation via high-watermark caching and skips namespaces with no active notification rules.

## Patterns

**Filter interface with IsNamespaceInScope + IsEntitlementInScope** — All filter implementations must satisfy filters.Filter (two methods). NamedFilter extends Filter with Name() string. Compile-time assertions enforce this: `var _ NamedFilter = (*HighWatermarkCache)(nil)`. (`var _ NamedFilter = (*HighWatermarkCache)(nil)`)
**Config struct with Validate() before construction** — Each filter with external dependencies takes a typed Config struct implementing models.Validator. NewXxx(cfg) calls cfg.Validate() as first step and returns error if invalid. (`var _ models.Validator = (*NotificationsFilterConfig)(nil)
func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**HighWatermark backend abstraction** — HighWatermarkCache delegates storage to a HighWatermarkBackend interface (Get/Record). In-memory LRU is the only current backend. New backends (e.g., Redis) must implement HighWatermarkBackend, not HighWatermarkCache itself. (`type HighWatermarkBackend interface { Get(...) (highWatermarkBackendGetResult, error); Record(...) error }`)
**CalculationTimeRecorder for post-calculation watermark updates** — Filters that track last calculation time implement CalculationTimeRecorder (extends Filter with RecordLastCalculation). The worker must call RecordLastCalculation after each successful recalculation. (`var _ CalculationTimeRecorder = (*HighWatermarkCache)(nil)`)
**LRU+TTL cache for external service lookups** — NotificationsFilter caches notification rules per namespace using lrux.CacheWithItemTTL to avoid per-event rule queries. New filters that call external services must apply the same caching strategy. (`ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Defines the Filter, NamedFilter, CalculationTimeRecorder interfaces and EntitlementFilterRequest/RecordLastCalculationRequest input types. All filter implementations start here. | EntitlementFilterRequest.Validate() must be called inside IsEntitlementInScope implementations — not doing so allows zero-value EventAt to pass through. |
| `highwatermark.go` | In-memory LRU cache preventing recalculation for events older than the last calculation timestamp. Reset operations always pass through (IsEntitlementInScope returns true for snapshot.ValueOperationReset). | defaultClockDrift (-1ms) is subtracted from watermark in the comparison — new time comparisons must replicate this offset or risk missing events under clock skew. |
| `notifications.go` | Skips recalculation for namespaces/entitlements with no active BalanceThreshold notification rules. Caches rule lookups with TTL. | An empty Features list in a rule means 'all features are in scope' — conditional logic must preserve this semantics. |

## Anti-Patterns

- Calling external services (notification, DB) inside IsEntitlementInScope without LRU+TTL caching — this runs per-event and must be low-latency
- Implementing a new filter without a compile-time interface assertion (var _ NamedFilter = ...)
- Skipping cfg.Validate() in a constructor — missing required dependencies will panic at runtime rather than fail at startup
- Modifying HighWatermarkCache to bypass the backend interface (breaks testability and future Redis backend)
- Adding mutable state to filter structs without concurrent-access protection — filters are called from parallel worker goroutines

## Decisions

- **Filters are composed as a chain rather than a single monolithic filter** — Each filter has a distinct concern (watermark vs notification rule existence) with different caching strategies. The chain allows the worker to short-circuit on the cheapest filter first.
- **HighWatermarkCache uses a pluggable backend interface even though only in-memory is implemented** — Future Redis-backed persistence would allow high-watermark state to survive worker restarts, preventing post-crash redundant recalculations.

## Example: Implement a new named filter that skips entitlements whose feature is archived

```
package filters

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
)

var _ NamedFilter = (*ArchivedFeatureFilter)(nil)

type ArchivedFeatureFilter struct {
	entSvc entitlement.Service
}

func (f *ArchivedFeatureFilter) Name() string { return "archived_feature" }
// ...
```

<!-- archie:ai-end -->
