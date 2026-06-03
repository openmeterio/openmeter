# lrux

<!-- archie:ai-start -->

> Generic LRU cache with per-item TTL and a synchronous Fetcher callback, wrapping hashicorp/golang-lru/v2. All time comparisons use pkg/clock (not time.Now) so tests can freeze time deterministically.

## Patterns

**Use clock.Now() never time.Now()** — All TTL comparisons use clock.Now() so tests can freeze time with clock.FreezeTime / clock.SetTime. Any new code here must follow this. (`if ok && (item.ExpiresAt.IsZero() || item.ExpiresAt.After(clock.Now())) {`)
**Functional options via cacheOptionsFunc** — Constructor accepts variadic cacheOptionsFunc. New options must define a func(*cacheOptions) and expose a With* factory like WithTTL. (`func WithTTL(ttl time.Duration) cacheOptionsFunc { return func(o *cacheOptions) { o.ttl = ttl } }`)
**Zero TTL means never-expire via zero time.Time** — When ttl == 0, ExpiresAt is the zero time.Time{}. item.ExpiresAt.IsZero() short-circuits to cache-hit, so ttl=0 is a never-expire signal, not zero-second expiry. (`// ttl=0 => ExpiresAt stays zero => item never expires`)
**Fetcher must not be nil — validate at construction** — Constructor returns an error if fetcher is nil; the package does no lazy validation. (`if fetcher == nil { return nil, fmt.Errorf("fetcher is required") }`)
**t.Context() in tests, defer clock.UnFreeze()** — All tests use t.Context() and defer clock.UnFreeze() immediately after clock.FreezeTime to prevent time leakage across parallel tests. (`clock.FreezeTime(baseTime); defer clock.UnFreeze(); item, err := cache.Get(t.Context(), "key")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lruitemttl.go` | Entire package — CacheWithItemTTL struct, Fetcher type, NewCacheWithItemTTL, Get, Refresh, fetchItem. | No mutex around fetch+store: concurrent Gets on an expired key issue multiple simultaneous fetcher calls. Intentional for simplicity but consider when the fetcher is expensive or has side effects. |
| `lruitemttl_test.go` | Unit tests validating TTL boundary behavior, cache hit/miss transitions, and error propagation from the fetcher. | Tests use clock.FreezeTime with baseTime parsed from RFC3339 — use the same approach for new TTL boundary tests to keep them deterministic. |

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
	"time"
	"github.com/openmeterio/openmeter/pkg/lrux"
)

cache, err := lrux.NewCacheWithItemTTL(100, func(ctx context.Context, key string) (string, error) {
	return fetchFromDB(ctx, key)
}, lrux.WithTTL(10*time.Second))
if err != nil { /* handle */ }
val, err := cache.Get(ctx, "mykey")
```

<!-- archie:ai-end -->
