# lrux

<!-- archie:ai-start -->

> Generic LRU cache with per-item TTL and a synchronous Fetcher callback, wrapping hashicorp/golang-lru/v2. Primary constraint: all time comparisons use pkg/clock (not time.Now) so tests can freeze time deterministically.

## Patterns

**Use clock.Now() never time.Now()** — All TTL comparisons inside the package use clock.Now() so tests can freeze time with clock.FreezeTime / clock.SetTime. Any new code added here must follow the same pattern. (`if ok && (item.ExpiresAt.IsZero() || item.ExpiresAt.After(clock.Now())) {`)
**Functional options via cacheOptionsFunc** — Constructor accepts variadic cacheOptionsFunc. New options must define a func(*cacheOptions) and expose a With* factory function following the existing WithTTL pattern. (`func WithTTL(ttl time.Duration) cacheOptionsFunc {
    return func(o *cacheOptions) { o.ttl = ttl }
}`)
**Zero TTL means never-expire via zero time.Time** — When ttl == 0, ExpiresAt is set to time.Time{} (zero value) via clock.Now().Add(0) logic path. The check item.ExpiresAt.IsZero() short-circuits to cache-hit, meaning ttl=0 is a never-expire signal — not zero-second expiry. (`// ttl=0 => ExpiresAt stays zero => item never expires`)
**Fetcher must not be nil — validate at construction** — Constructor returns an error if fetcher is nil. Callers must provide a non-nil fetcher; the package does not attempt lazy validation. (`if fetcher == nil { return nil, fmt.Errorf("fetcher is required") }`)
**t.Context() in tests, defer clock.UnFreeze()** — All tests use t.Context() instead of context.Background() and defer clock.UnFreeze() immediately after clock.FreezeTime to prevent time leakage across parallel tests. (`clock.FreezeTime(baseTime)
defer clock.UnFreeze()
item, err := cache.Get(t.Context(), "key")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lruitemttl.go` | Entire package implementation — CacheWithItemTTL struct, Fetcher type, NewCacheWithItemTTL, Get, Refresh, fetchItem. | No mutex around fetch+store: concurrent Gets on an expired key will issue multiple simultaneous fetcher calls. This is intentional for simplicity but must be considered when the fetcher is expensive or has side effects. |
| `lruitemttl_test.go` | Unit tests validating TTL boundary behavior, cache hit/miss transitions, and error propagation from the fetcher. | Tests use clock.FreezeTime with baseTime parsed from RFC3339 — use the same approach for any new TTL boundary tests to keep them deterministic. |

## Anti-Patterns

- Using time.Now() inside a Fetcher or anywhere in this package — always use clock.Now()
- Adding a mutex around Get without profiling — stampede tolerance is an intentional design choice
- Constructing with a nil fetcher — the constructor rejects it but upstream callers should validate early
- Using context.Background() in tests when t.Context() is available

## Decisions

- **Wrap hashicorp/golang-lru with CacheItemWithTTL instead of using lru's built-in expiry** — lru's native eviction does not integrate with pkg/clock, making TTL boundary tests non-deterministic. Storing ExpiresAt on the item and checking via clock.Now() gives full test control.

## Example: Create a cache that fetches string values by string key with a 10-second TTL

```
import (
    "context"
    "github.com/openmeterio/openmeter/pkg/lrux"
    "time"
)

cache, err := lrux.NewCacheWithItemTTL(100, func(ctx context.Context, key string) (string, error) {
    return fetchFromDB(ctx, key)
}, lrux.WithTTL(10*time.Second))
if err != nil { /* handle */ }
val, err := cache.Get(ctx, "mykey")
```

<!-- archie:ai-end -->
