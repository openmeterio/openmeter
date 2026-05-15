# filters

<!-- archie:ai-start -->

> Pluggable filter chain that gates whether a RecalculateEvent should be processed for a given namespace/entitlement. Prevents redundant ClickHouse queries via high-watermark caching and short-circuits namespaces with no active notification rules.

## Patterns

**Filter / NamedFilter / CalculationTimeRecorder interface hierarchy** — All implementations must satisfy Filter (IsNamespaceInScope + IsEntitlementInScope). Named filters also implement Name() string. Filters that update state after calculation implement CalculationTimeRecorder. All enforce with compile-time var _ assertions. (`var _ NamedFilter = (*HighWatermarkCache)(nil)
var _ CalculationTimeRecorder = (*HighWatermarkCache)(nil)`)
**Config struct with Validate() called first in constructor** — Each filter with external dependencies takes a typed Config struct implementing models.Validator. The constructor calls cfg.Validate() as its first statement and returns error on failure — never panics. (`func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) {
    if err := cfg.Validate(); err != nil { return nil, err }
    ...
}`)
**HighWatermarkBackend abstraction for storage** — HighWatermarkCache delegates all storage to HighWatermarkBackend interface (Get/Record). Only in-memory LRU is implemented now. New backends (Redis) must implement HighWatermarkBackend — never add storage directly to HighWatermarkCache. (`type HighWatermarkBackend interface {
    Get(ctx context.Context, entitlementID string) (highWatermarkBackendGetResult, error)
    Record(ctx context.Context, req RecordLastCalculationRequest) error
}`)
**LRU+TTL cache for all external service lookups** — Any filter that calls an external service (notification, DB) must cache results using lrux.CacheWithItemTTL — filters run per-event on the hot worker path and must not issue unbounded external calls. (`ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))`)
**Reset operations always pass high-watermark filter** — snapshot.ValueOperationReset must bypass the watermark check and return true unconditionally — fresh notification events for new periods must always flow through. (`if req.Operation == snapshot.ValueOperationReset { return true, nil }`)
**EntitlementFilterRequest.Validate() inside every IsEntitlementInScope** — All IsEntitlementInScope implementations must call req.Validate() first and return the error. Zero-value EventAt silently bypasses watermark comparisons if not validated. (`func (f *Foo) IsEntitlementInScope(ctx context.Context, req EntitlementFilterRequest) (bool, error) {
    if err := req.Validate(); err != nil { return false, err }
    ...
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Defines the Filter, NamedFilter, CalculationTimeRecorder interfaces and the EntitlementFilterRequest / RecordLastCalculationRequest input types. | EntitlementFilterRequest.Validate() is the canonical guard — all implementations must call it. Adding fields here is a breaking change for all filter implementations. |
| `highwatermark.go` | In-memory LRU cache preventing recalculation for events older than the last successful calculation timestamp. | defaultClockDrift (-1ms) is subtracted from the watermark in comparisons — replicate this offset in any new time comparison or events under clock skew will be dropped. HighWatermarkInMemoryBackend.cache is not goroutine-safe without lru's built-in locking — do not bypass it. |
| `notifications.go` | Skips recalculation for namespaces/entitlements with no active BalanceThreshold notification rules. Caches rule lookups with TTL via lrux. | Empty Features list in a BalanceThreshold rule means 'all features in scope' — any new feature-matching logic must preserve this semantics. NotificationsFilter does not implement CalculationTimeRecorder (no RecordLastCalculation) — do not add watermark logic here. |

## Anti-Patterns

- Calling external services inside IsEntitlementInScope without LRU+TTL caching — runs per-event and will saturate downstream services
- Implementing a new filter without a compile-time var _ interface assertion
- Skipping cfg.Validate() in a new filter constructor — nil dependencies cause nil-pointer panics at runtime rather than startup
- Adding mutable state to filter structs without concurrent-access protection — filters are called concurrently from worker goroutines
- Hardcoding storage into HighWatermarkCache instead of implementing a new HighWatermarkBackend

## Decisions

- **Filters composed as a named chain rather than a single monolithic gating function** — Each filter has a distinct concern (watermark vs notification rule existence) with different caching strategies. The chain allows the worker to short-circuit on the cheapest filter (namespace-level) before the more expensive entitlement-level check.
- **HighWatermarkCache uses a pluggable HighWatermarkBackend interface despite only in-memory being implemented** — Future Redis-backed persistence would allow high-watermark state to survive worker restarts, eliminating post-crash redundant ClickHouse queries across all entitlements.

## Example: Add a new NamedFilter that skips entitlements whose feature is archived

```
package filters

import (
    "context"
    "fmt"

    "github.com/openmeterio/openmeter/openmeter/entitlement"
    "github.com/openmeterio/openmeter/pkg/models"
)

var _ NamedFilter = (*ArchivedFeatureFilter)(nil)

type ArchivedFeatureFilterConfig struct {
    EntitlementService entitlement.Service
}
// ...
```

<!-- archie:ai-end -->
