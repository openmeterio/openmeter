# clock

<!-- archie:ai-start -->

> Process-global test clock replacing time.Now() with a controllable source backed by atomic drift/freeze vars. All production code must call clock.Now() so tests can freeze or shift time deterministically without interface injection.

## Patterns

**clock.Now() everywhere** — Production code must call clock.Now() instead of time.Now(). Package-level atomics mean no interface injection. (`import "github.com/openmeterio/openmeter/pkg/clock"; t := clock.Now()`)
**FreezeTime/UnFreeze in tests** — Freeze with clock.FreezeTime(t) and restore with defer clock.UnFreeze() or defer clock.ResetTime(). (`clock.FreezeTime(testutils.GetRFC3339Time(t, "2024-06-30T15:39:00Z")); defer clock.UnFreeze()`)
**Monotonic clock stripped** — clock.Now() always calls .Round(0) to strip the monotonic reading, preventing subtle comparison bugs when storing/comparing times. (`return time.Now().Add(-driftDuration).Round(0)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `clock.go` | Entire package: atomic drift int64 for SetTime, atomic frozen int32 + atomic.Value for FreezeTime. SetTime computes drift as time.Since(t).Nanoseconds(). | drift is the offset subtracted from wall time, not an absolute time. FreezeTime and SetTime are independent mechanisms — FreezeTime short-circuits drift entirely. |

## Anti-Patterns

- Calling time.Now() directly in production code — always use clock.Now().
- Forgetting defer clock.ResetTime() / clock.UnFreeze() in tests — leaks frozen/drifted state across cases.
- Adding a context parameter or Clock interface — the global atomic design is intentional for zero-overhead usage.

## Decisions

- **Package-level atomics instead of an injected Clock interface.** — Zero call-site changes across the codebase; atomic ops are lock-free and safe for concurrent test use.

<!-- archie:ai-end -->
