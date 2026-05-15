# timeutil

<!-- archie:ai-start -->

> Shared time-interval primitives (ClosedPeriod, OpenPeriod, StartBoundedPeriod, Recurrence, Timeline) used throughout billing, subscription, entitlement, and metering to reason about billable windows, recurrence schedules, and event timelines with explicit boundary semantics.

## Patterns

**Three Contains variants with documented boundary semantics — pick the right one** — Every period type implements Period: Contains (inclusive start, exclusive end — the default for billing windows), ContainsInclusive (both inclusive), ContainsExclusive (both exclusive). Mixing them silently changes charge window edges. (`period.Contains(t)          // [from, to) — default billing window
period.ContainsInclusive(t) // [from, to] — use for point-in-time snapshots`)
**Recurrence is anchor-based; always construct via NewRecurrence or NewRecurrenceFromISODuration** — Anchor can be any past or future point; iteration walks forward/backward from it. Constructors call Validate() — direct struct construction skips validation and may pass zero anchor or non-positive interval. (`rec, err := timeutil.NewRecurrenceFromISODuration(datetime.DurationMonth, anchorTime)
period, err := rec.GetPeriodAt(now)`)
**Boundary enum controls inclusive/exclusive edge of Recurrence iteration** — IterateFromNextAfter and IterateFromPrevBefore accept a Boundary argument. Inclusive returns t if it falls on a boundary; Exclusive skips to next/prev. Always use constants timeutil.Inclusive or timeutil.Exclusive. (`next, err := rec.NextAfter(t, timeutil.Exclusive)`)
**OpenPeriod uses *time.Time — nil means open/unbounded; touching periods do NOT overlap** — Nil From means open start; nil To means open end. Touching periods that share only an endpoint return nil from Intersection — this is intentional: [t,t] has zero length. (`// OpenPeriod{From: &t1, To: nil} extends to infinity
// Intersection of [before,now] and [now,after] returns nil (they touch, not overlap)`)
**Use ISODuration for month/year recurrences, never time.Duration arithmetic** — Month and year lengths vary; time.Duration math gives wrong billing period ends for variable-length months. RecurrencePeriodMonth/Year constants wrap ISODuration correctly. (`// Wrong: anchor.Add(30 * 24 * time.Hour) -- wrong for February
// Correct: timeutil.NewRecurrence(timeutil.RecurrencePeriodMonth, anchor)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `period.go` | Period interface requiring all three Contains variants. All concrete types assert var _ Period = MyType{} at declaration. | Any new period type must implement all three methods; missing one causes compile errors at assertion sites. |
| `closedperiod.go` | Fully-bounded [from, to] interval with Overlaps, OverlapsInclusive, Intersection, Truncate, Equal. | Overlaps returns false for exactly-touching periods ([1,2],[2,3]); use OverlapsInclusive when sequential billing periods must be treated as adjacent. |
| `openperiod.go` | Half or fully-open interval with Intersection, Difference, Union, IsSupersetOf using *time.Time pointers. | Intersection of touching OpenPeriods (sharing one endpoint) returns nil — intentional. Union with either period being fully open ({nil,nil}) propagates openness. |
| `recurrence.go` | Recurrence schedule with Anchor+Interval, GetPeriodAt, IterateFromNextAfter, IterateFromPrevBefore, and RecurrenceIterator for bidirectional stepping. | MAX_SAFE_ITERATIONS = 1_000_000 cap prevents infinite loops; fractional ISO durations cause addIntervalNTimes to error. Backward iteration past 1733 is not supported (explicit TODO). |
| `timeline.go` | Generic sorted Timeline[T] of Timed[T] values with GetClosedPeriods (n-1 segments from n timestamps) and GetOpenPeriods (n+1 segments including open ends). | GetClosedPeriods with a single timestamp returns one zero-length period (From==To). GetOpenPeriods always returns len(times)+1 segments. NewTimeline sorts input — insertion order does not matter. |
| `boundedperiod.go` | StartBoundedPeriod with mandatory From and optional *time.Time To; implements Period. | ContainsExclusive treats From as exclusive start — different from ClosedPeriod.ContainsExclusive which has inclusive start via Contains. |

## Anti-Patterns

- Using time.Duration arithmetic instead of ISODuration for month/year recurrences — month lengths vary and Duration math produces wrong billing period ends
- Constructing Recurrence{} struct literal without calling NewRecurrence — skips Validate(), allowing zero anchor or non-positive interval to pass through silently
- Assuming Overlaps and OverlapsInclusive are interchangeable — sequential billing periods must use Overlaps (returns false for touching) to avoid double-counting
- Constructing OpenPeriod{From: &t, To: &t} expecting a valid point interval — Intersection treats same-start-and-end as zero-length and returns nil

## Decisions

- **Contains uses half-open [from, to) semantics as the default** — Half-open intervals compose without overlap for sequential billing windows: period N ends at T, period N+1 starts at T, Contains(T) is true for N+1 only, preventing double-counting.
- **Recurrence iterates from a floating anchor rather than an epoch** — Subscription anchors can be any date; billing periods must align to the original subscription start regardless of when the calculation runs.

## Example: Find the billing period containing a given usage timestamp using monthly recurrence

```
import (
    "time"
    "github.com/openmeterio/openmeter/pkg/timeutil"
    "github.com/openmeterio/openmeter/pkg/datetime"
)

anchor := subscriptionStart
rec, err := timeutil.NewRecurrenceFromISODuration(datetime.DurationMonth, anchor)
if err != nil { return err }
period, err := rec.GetPeriodAt(usageTime)
if err != nil { return err }
// period.From <= usageTime < period.To (half-open, no double-counting)
```

<!-- archie:ai-end -->
