# lrux

<!-- archie:ai-start -->

> Generic LRU cache wrapper with per-item TTL on top of hashicorp/golang-lru/v2, fetching missing or expired entries through a supplied Fetcher. Used by entitlement balanceworker (and its filters) for cached lookups.

## Patterns

**Generic CacheWithItemTTL[K,V] embedding lru.Cache** — The cache embeds `*lru.Cache[K, CacheItemWithTTL[V]]` and stores values wrapped with an ExpiresAt timestamp; K is `comparable`, V is `any`. (`type CacheWithItemTTL[K comparable, V any] struct { *lru.Cache[K, CacheItemWithTTL[V]]; fetcher Fetcher[K, V]; ttl time.Duration }`)
**Constructor validates fetcher and ttl** — `NewCacheWithItemTTL(size, fetcher, opts...)` returns an error if fetcher is nil or ttl < 0; TTL is set via the `WithTTL(d)` functional option (default 0 = never expires). (`cache, err := NewCacheWithItemTTL(10, fetchFn, WithTTL(time.Second*10))`)
**Clock-driven expiry via pkg/clock** — Expiry is evaluated against `clock.Now()` (not time.Now), so tests can FreezeTime/SetTime. Get returns the cached value when ExpiresAt is zero or after now; otherwise it fetches and stores. Refresh always re-fetches. (`if ok && (item.ExpiresAt.IsZero() || item.ExpiresAt.After(clock.Now())) { return item.Value, nil }`)
**No locking around fetch** — Get/Refresh do not hold a mutex during fetch (explicit code comment), so concurrent misses may fetch the same key multiple times — acceptable by design. (`// NOTE: we are not using a mutex here, as we don't want to limit the number of fetches for now`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lruitemttl.go` | CacheWithItemTTL implementation: NewCacheWithItemTTL, Get, Refresh, internal fetchItem; Fetcher type and WithTTL option. | Expiry uses clock.Now(); never substitute time.Now() or tests using FreezeTime will be bypassed. ttl=0 means entries never expire. |
| `lruitemttl_test.go` | Demonstrates the clock.FreezeTime/SetTime + defer clock.UnFreeze testing pattern and Get/Refresh error propagation. | Pairs FreezeTime with defer UnFreeze — replicate this when adding tests so frozen time does not leak. |

## Anti-Patterns

- Using time.Now() instead of clock.Now() for expiry, breaking deterministic tests.
- Assuming Get is single-flight — concurrent misses can trigger duplicate fetches by design.
- Passing a nil fetcher or negative ttl; the constructor rejects both with an error.

## Decisions

- **Per-item TTL stored alongside the value (CacheItemWithTTL) rather than relying on a global cache eviction policy.** — Allows lazy refresh-on-read: expired items are re-fetched on Get while still bounded by LRU size.
- **No mutex around fetch.** — Deliberately avoids throttling fetch concurrency; duplicate fetches are tolerated to keep the cache simple and lock-free.

## Example: Build a TTL cache backed by a fetch function

```
import (
	"github.com/openmeterio/openmeter/pkg/lrux"
)

cache, err := lrux.NewCacheWithItemTTL(
	1000,
	func(ctx context.Context, id string) (Entity, error) { return repo.Get(ctx, id) },
	lrux.WithTTL(30*time.Second),
)
if err != nil {
	return err
}
v, err := cache.Get(ctx, id)
```

<!-- archie:ai-end -->
