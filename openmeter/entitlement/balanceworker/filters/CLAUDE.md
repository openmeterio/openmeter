# filters

<!-- archie:ai-start -->

> Provides the scoping layer that decides whether a given entitlement/namespace should be recalculated and snapshot-emitted by the balance worker. Defines the Filter interface and two implementations: HighWatermarkCache (dedup by last-calculation time) and NotificationsFilter (only entitlements covered by a balance-threshold notification rule).

## Patterns

**Filter interface hierarchy** — Filter declares IsNamespaceInScope + IsEntitlementInScope. NamedFilter embeds Filter + Name(); CalculationTimeRecorder embeds Filter + RecordLastCalculation. Implementations pick the interface that matches their capability and assert it. (`var _ NamedFilter = (*HighWatermarkCache)(nil); var _ CalculationTimeRecorder = (*HighWatermarkCache)(nil)`)
**Validate request before scoping** — Both IsEntitlementInScope implementations call req.Validate() first; EntitlementFilterRequest.Validate collects id/namespace/operation/eventAt errors via errors.Join. (`if err := req.Validate(); err != nil { return false, err }`)
**Backend-abstracted cache** — HighWatermarkCache delegates to a HighWatermarkBackend interface (Get/Record); HighWatermarkInMemoryBackend wraps hashicorp golang-lru/v2. Swap the backend, not the cache logic. (`func NewHighWatermarkCache(size int) (*HighWatermarkCache, error) { backend, err := NewHighWatermarkInMemoryBackend(size); ... }`)
**Config struct with Validate constructor guard** — NotificationsFilter is built from NotificationsFilterConfig (implements models.Validator); NewNotificationsFilter calls cfg.Validate() before constructing. Required deps (NotificationService) and positive TTL/size are enforced there. (`func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**TTL'd rule cache keyed by namespace** — NotificationsFilter caches rules per namespace via lrux.CacheWithItemTTL with a loader (fetchRulesForNamespace) and lrux.WithTTL; it only fetches EventTypeBalanceThreshold rules. (`ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Declares the Filter / NamedFilter / CalculationTimeRecorder interfaces and the EntitlementFilterRequest + RecordLastCalculationRequest input structs. | EntitlementFilterRequest.Operation is a snapshot.ValueOperationType; Validate requires Entitlement.ID, Namespace, a valid Operation, and a non-zero EventAt. |
| `highwatermark.go` | Dedup filter: keeps the last CalculatedAt per entitlement; an event is in-scope only if its EventAt is after (highWatermark - defaultClockDrift). | Reset operations (snapshot.ValueOperationReset) always return in-scope. Missing cache entries are treated as in-scope (fail-open); deleted entitlements are out-of-scope regardless of watermark. defaultClockDrift is 1ms to tolerate NTP skew between worker nodes. |
| `notifications.go` | Scope filter that limits recalculation to entitlements matching a balance-threshold notification rule's feature list (empty Features list => all features in scope). | Matches both FeatureKey and FeatureID against rule.Config.BalanceThreshold.Features; rules with a nil BalanceThreshold are skipped. IsNamespaceInScope returns true only if the namespace has >0 such rules. |

## Anti-Patterns

- Querying NotificationService directly per event instead of going through the TTL'd ruleCache.
- Skipping req.Validate() in a Filter implementation before making a scope decision.
- Making HighWatermarkCache fail-closed on cache miss — current contract is fail-open (treat unknown entitlements as in-scope).
- Hardcoding a new cache backend inside HighWatermarkCache instead of implementing HighWatermarkBackend.
- Treating reset events as dedup-eligible — they must always pass so the new period gets a fresh snapshot for notifications.

## Decisions

- **High-watermark dedup compares against highWatermark minus a 1ms defaultClockDrift.** — Multiple worker nodes may have slightly different clocks; allowing 1ms of drift (guaranteed on AWS/GCP NTP) prevents valid events from being dropped at the boundary.
- **NotificationsFilter only loads EventTypeBalanceThreshold rules and matches feature key or id.** — The balance worker only needs to recompute snapshots that can fire a balance-threshold notification; filtering at the namespace/feature level avoids recomputing entitlements no rule cares about.
- **Cache logic and storage are split via HighWatermarkBackend.** — Lets the in-memory LRU be swapped for a distributed backend without changing the in-scope decision logic.

## Example: Building a NamedFilter from a validated config with a TTL'd per-namespace loader cache

```
func NewNotificationsFilter(cfg NotificationsFilterConfig) (NamedFilter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	filter := &NotificationsFilter{notificationService: cfg.NotificationService}
	ruleCache, err := lrux.NewCacheWithItemTTL(cfg.CacheSize, filter.fetchRulesForNamespace, lrux.WithTTL(cfg.CacheTTL))
	if err != nil {
		return nil, err
	}
	filter.ruleCache = ruleCache
	return filter, nil
}
```

<!-- archie:ai-end -->
