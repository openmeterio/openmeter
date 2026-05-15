# clock

<!-- archie:ai-start -->

> Process-global test clock that replaces time.Now() with a controllable time source backed by atomic drift/freeze vars. All production code must call clock.Now() so tests can freeze or shift time deterministically without any interface injection.

## Patterns

**clock.Now() everywhere** — Production code must call clock.Now() instead of time.Now(). The package uses package-level atomics so no interface injection is needed. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**FreezeTime/UnFreeze in tests** — Tests freeze the clock with clock.FreezeTime(t) and restore with defer clock.UnFreeze() or defer clock.ResetTime(). (`clock.FreezeTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z")); defer clock.UnFreeze()`)
**Monotonic clock stripped** — clock.Now() always calls .Round(0) to strip the monotonic reading, preventing subtle comparison bugs when storing/comparing times. (`return time.Now().Add(-driftDuration).Round(0)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `clock.go` | Entire package: atomic drift int64 for SetTime, atomic frozen int32 + atomic.Value for FreezeTime. SetTime computes drift as time.Since(t).Nanoseconds(). | drift is the offset subtracted from wall time, not an absolute time value. FreezeTime and SetTime are two independent mechanisms — FreezeTime short-circuits drift entirely. |

## Anti-Patterns

- Calling time.Now() directly in production code — always use clock.Now()
- Forgetting defer clock.ResetTime() or defer clock.UnFreeze() in tests, which leaks frozen/drifted state across test cases
- Adding a context parameter or Clock interface — the global atomic design is intentional for zero-overhead usage across the codebase

## Decisions

- **Package-level atomics instead of an injected Clock interface** — Zero call-site changes needed across the codebase; atomic ops are lock-free and safe for concurrent test use.

<!-- archie:ai-end -->
