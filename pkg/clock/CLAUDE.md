# clock

<!-- archie:ai-start -->

> Mockable global clock so time-dependent code (billing periods, entitlement resets, subscription cadences) can be deterministically controlled in tests. A massive dependency magnet (~100 in-edges) — every package that reads 'current time' should call clock.Now() rather than time.Now().

## Patterns

**Read time via clock.Now()** — Production code that needs the current time must call clock.Now(), never time.Now() directly, so tests can freeze or drift the clock. (`now := clock.Now()`)
**Atomic global state, no struct** — The clock is package-level mutable state (drift, frozen, frozenTime) guarded by sync/atomic. There is no Clock instance to inject; all access is via package functions. (`atomic.StoreInt32(&frozen, 1); frozenTime.Store(t)`)
**Strip monotonic reading** — Now() calls .Round(0) on returned times to remove the monotonic clock reading, keeping wall-clock-only timestamps for stable comparisons/serialization. (`return t.Round(0)`)
**Freeze must be paired with UnFreeze** — Tests calling FreezeTime(t) must defer UnFreeze() in the same scope so frozen time does not leak into later subtests (project-wide rule in AGENTS.md). (`clock.FreezeTime(t); defer clock.UnFreeze()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `clock.go` | Entire implementation: Now, SetTime (drift-based offset), ResetTime, FreezeTime (hard pin), UnFreeze. | SetTime sets a relative drift (clock keeps advancing), FreezeTime pins an absolute instant (clock stops). They are different mechanisms — pair SetTime with ResetTime, FreezeTime with UnFreeze. |
| `clock_test.go` | Demonstrates SetTime/ResetTime usage with testutils.GetRFC3339Time. | Because drift mode keeps ticking, assertions use a tolerance (diff < time.Second) rather than exact equality. |

## Anti-Patterns

- Calling time.Now() in production code instead of clock.Now() — breaks deterministic tests.
- FreezeTime without a deferred UnFreeze, leaking frozen time into other tests.
- Adding a struct-based Clock or injecting a clock instance — this package is intentionally global package-level state.
- Mixing SetTime/FreezeTime modes without resetting the previous one.

## Decisions

- **Global package-level clock guarded by sync/atomic rather than dependency injection.** — Avoids threading a Clock interface through every constructor in a ~100-importer codebase; atomics keep it goroutine-safe for parallel tests.
- **Two override modes: relative drift (SetTime) and absolute freeze (FreezeTime).** — Drift lets time still advance for realistic flows; freeze pins an exact instant for precise period/boundary assertions.

## Example: Deterministically pin current time in a test

```
import "github.com/openmeterio/openmeter/pkg/clock"

clock.FreezeTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z"))
defer clock.UnFreeze()
now := clock.Now()
```

<!-- archie:ai-end -->
