# clock

<!-- archie:ai-start -->

> Process-global test clock that replaces time.Now() calls with a controllable time source. All production code must call clock.Now() instead of time.Now() so tests can freeze or shift time deterministically.

## Patterns

**clock.Now() everywhere** — Production code must call clock.Now() not time.Now(). The package uses package-level atomics so no injection is needed. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**FreezeTime/UnFreeze in tests** — Tests freeze the clock with clock.FreezeTime(t) and unfreeze with defer clock.UnFreeze() or defer clock.ResetTime() to restore drift-based mode. (`clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z")); defer clock.ResetTime()`)
**Monotonic clock stripped** — clock.Now() always calls .Round(0) to strip the monotonic reading, preventing subtle comparison bugs when storing/comparing times. (`return time.Now().Add(-driftDuration).Round(0)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `clock.go` | Entire package implementation: atomic drift int64 for SetTime, atomic frozen int32 + atomic.Value for FreezeTime. | SetTime computes drift as time.Since(t).Nanoseconds() — drift is the offset to subtract from wall time, not the absolute time. |

## Anti-Patterns

- Calling time.Now() directly in production code — always use clock.Now()
- Forgetting defer clock.ResetTime() or defer clock.UnFreeze() in tests, which leaks frozen state across test cases
- Adding a context parameter or interface indirection — the global var design is intentional for zero-overhead usage

## Decisions

- **Package-level atomics instead of an injected Clock interface** — Zero call-site changes needed across the codebase; atomic ops are lock-free and safe for concurrent test use.

<!-- archie:ai-end -->
