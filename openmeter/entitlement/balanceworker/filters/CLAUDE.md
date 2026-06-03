# filters

<!-- archie:ai-start -->

> Pluggable filter chain that gates whether a RecalculateEvent should be processed for a given namespace/entitlement. Prevents redundant ClickHouse queries via high-watermark caching and short-circuits namespaces with no active notification rules.

## Patterns

**Filter / NamedFilter / CalculationTimeRecorder interface hierarchy** — All implementations satisfy Filter (IsNamespaceInScope + IsEntitlementInScope). Named filters also implement Name() string. Filters that update state after calculation implement CalculationTimeRecorder. Enforce with compile-time var _ assertions. (`var _ NamedFilter = (*HighWatermarkCache)(nil)
var _ CalculationTimeRecorder = (*HighWatermarkCache)(nil)`)
**Config struct with Validate() called first in constructor** — Each filter with external dependencies takes a typed Config struct implementing models.Validator; the constructor calls cfg.Validate() as its first statement and returns error on failure — never panics. (`func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**HighWatermarkBackend abstraction for storage** — HighWatermarkCache delegates all storage to the HighWatermarkBackend interface (Get/Record). Only in-memory LRU is implemented; new backends (Redis) must implement HighWatermarkBackend — never add storage directly to HighWatermarkCache. (`type HighWatermarkBackend interface { Get(ctx, entitlementID string) (highWatermarkBackendGetResult, error); Record(ctx, req RecordLastCalculationRequest) error }`)
**LRU+TTL cache for all external service lookups** — Any filter calling an external service (notification, DB) must cache results using lrux.CacheWithItemTTL — filters run per-event on the hot worker path and must not issue unbounded external calls. (`ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))`)
**Reset operations always pass high-watermark filter** — snapshot.ValueOperationReset must bypass the watermark check and return true unconditionally — fresh notification events for new periods must always flow through. (`if req.Operation == snapshot.ValueOperationReset { return true, nil }`)
**EntitlementFilterRequest.Validate() inside every IsEntitlementInScope** — All IsEntitlementInScope implementations must call req.Validate() first and return the error. A zero-value EventAt silently bypasses watermark comparisons if not validated. (`func (f *Foo) IsEntitlementInScope(ctx context.Context, req EntitlementFilterRequest) (bool, error) { if err := req.Validate(); err != nil { return false, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Defines Filter, NamedFilter, CalculationTimeRecorder interfaces and the EntitlementFilterRequest / RecordLastCalculationRequest input types. | EntitlementFilterRequest.Validate() is the canonical guard — all implementations must call it. Adding fields here is a breaking change for all filter implementations. |
| `highwatermark.go` | In-memory LRU cache preventing recalculation for events older than the last successful calculation timestamp. | defaultClockDrift (-1ms) is subtracted from the watermark in comparisons — replicate this offset in any new time comparison or events under clock skew will be dropped. Do not bypass the LRU's built-in locking on HighWatermarkInMemoryBackend.cache. |
| `notifications.go` | Skips recalculation for namespaces/entitlements with no active BalanceThreshold notification rules; caches rule lookups with TTL via lrux. | Empty Features list in a BalanceThreshold rule means 'all features in scope' — preserve this semantics. NotificationsFilter does not implement CalculationTimeRecorder — do not add watermark logic here. |

## Anti-Patterns

- Calling external services inside IsEntitlementInScope without LRU+TTL caching — runs per-event and saturates downstream services
- Implementing a new filter without a compile-time var _ interface assertion
- Skipping cfg.Validate() in a new filter constructor — nil dependencies cause runtime nil-pointer panics rather than startup errors
- Adding mutable state to filter structs without concurrent-access protection — filters are called concurrently from worker goroutines
- Hardcoding storage into HighWatermarkCache instead of implementing a new HighWatermarkBackend

## Decisions

- **Filters composed as a named chain rather than a single monolithic gating function** — Each filter has a distinct concern (watermark vs notification rule existence) with different caching strategies. The chain lets the worker short-circuit on the cheapest filter (namespace-level) before the more expensive entitlement-level check.
- **HighWatermarkCache uses a pluggable HighWatermarkBackend interface despite only in-memory being implemented** — Future Redis-backed persistence would let high-watermark state survive worker restarts, eliminating post-crash redundant ClickHouse queries across all entitlements.

## Example: Add a new NamedFilter that skips entitlements whose feature is archived

```
package filters

import (
    "context"
    "github.com/openmeterio/openmeter/openmeter/entitlement"
    "github.com/openmeterio/openmeter/pkg/models"
)

var _ NamedFilter = (*ArchivedFeatureFilter)(nil)

type ArchivedFeatureFilterConfig struct { EntitlementService entitlement.Service }

func (f *ArchivedFeatureFilter) IsEntitlementInScope(ctx context.Context, req EntitlementFilterRequest) (bool, error) {
    if err := req.Validate(); err != nil { return false, err }
    // cached lookup of feature archived state, return false to skip
// ...
```

<!-- archie:ai-end -->
