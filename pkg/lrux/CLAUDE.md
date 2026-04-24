# lrux

<!-- archie:ai-start -->

> Generic LRU cache with per-item TTL and a synchronous Fetcher callback, wrapping hashicorp/golang-lru/v2. Primary constraint: all time comparisons use pkg/clock (not time.Now) so tests can freeze time.

## Patterns

**Clock package for time** — All calls to time.Now() are replaced with clock.Now() so the cache TTL logic is testable with clock.FreezeTime / clock.SetTime. (`item.ExpiresAt.After(clock.Now())`)
**Functional options for construction** — Constructor accepts variadic cacheOptionsFunc; add new options by defining a func(*cacheOptions) and exposing a With* factory function. (`func WithTTL(ttl time.Duration) cacheOptionsFunc { return func(o *cacheOptions) { o.ttl = ttl } }`)
**Zero TTL means no expiry** — When ttl == 0, ExpiresAt is set to clock.Now().Add(0) == Now, but the check is item.ExpiresAt.IsZero() || After(clock.Now()), so a truly zero ExpiresAt (time.Time{}) means never-expire — set ttl=0 intentionally for this behavior. (`if ok && (item.ExpiresAt.IsZero() || item.ExpiresAt.After(clock.Now())) {`)
**Fetcher must not be nil** — Constructor returns an error if fetcher is nil; always validate the fetcher before calling NewCacheWithItemTTL. (`if fetcher == nil { return nil, fmt.Errorf("fetcher is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lruitemttl.go` | Entire package implementation — CacheWithItemTTL, Fetcher type, NewCacheWithItemTTL, Get, Refresh. | No mutex around fetch+store; concurrent Gets on an expired key will issue multiple fetcher calls. This is documented as intentional but must be considered for expensive fetchers. |
| `lruitemttl_test.go` | Unit tests using clock freeze to validate TTL boundary behavior and error propagation. | Tests use t.Context() (not context.Background()) and defer clock.UnFreeze() — new tests must follow the same pattern. |

## Anti-Patterns

- Using time.Now() inside a Fetcher or anywhere in this package — always use clock.Now().
- Adding a mutex around Get without benchmarking — the current design intentionally allows stampede for simplicity.
- Constructing with a nil fetcher — the constructor rejects it but callers should validate early.

## Decisions

- **Wrap hashicorp/golang-lru with CacheItemWithTTL instead of using lru's built-in expiry** — Allows TTL checks to go through pkg/clock for deterministic test control, which lru's native eviction does not support.

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
if err != nil { ... }
val, err := cache.Get(ctx, "mykey")
```

<!-- archie:ai-end -->
