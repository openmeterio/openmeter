# timeutil

<!-- archie:ai-start -->

> Shared time-interval primitives (ClosedPeriod, OpenPeriod, StartBoundedPeriod, Recurrence, Timeline) used across billing, subscription, entitlement, and metering to reason about billable windows, recurrence schedules, and event timelines with explicit boundary semantics.

## Patterns

**Three Contains variants with documented boundary semantics** — Every period implements Period: Contains (inclusive start, exclusive end — default billing window), ContainsInclusive (both inclusive), ContainsExclusive (both exclusive). Mixing them silently shifts charge-window edges. (`period.Contains(t)          // [from, to) — default
period.ContainsInclusive(t) // [from, to] — point-in-time snapshots`)
**Recurrence is anchor-based; construct via constructors** — Always use NewRecurrence or NewRecurrenceFromISODuration — they call Validate(); direct struct construction skips validation and may pass a zero anchor or non-positive interval. (`rec, err := timeutil.NewRecurrenceFromISODuration(datetime.DurationMonth, anchor); period, err := rec.GetPeriodAt(now)`)
**Boundary enum controls iteration edge** — IterateFromNextAfter/IterateFromPrevBefore take a Boundary; Inclusive returns t when on a boundary, Exclusive skips. Use the constants timeutil.Inclusive / timeutil.Exclusive. (`next, err := rec.NextAfter(t, timeutil.Exclusive)`)
**OpenPeriod uses *time.Time; touching periods do NOT overlap** — nil From = open start, nil To = open end. Periods sharing only an endpoint return nil from Intersection — intentional, since [t,t] has zero length. (`// Intersection of [before,now] and [now,after] returns nil (touch, not overlap)`)
**Use ISODuration for month/year recurrences** — Month/year lengths vary; time.Duration math gives wrong period ends. Use RecurrencePeriodMonth/Year (wrapping ISODuration) instead. (`// Wrong: anchor.Add(30 * 24 * time.Hour)
// Correct: timeutil.NewRecurrence(timeutil.RecurrencePeriodMonth, anchor)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `period.go` | Period interface requiring all three Contains variants; concrete types assert var _ Period = MyType{}. | A new period type must implement all three methods or assertion sites fail to compile. |
| `closedperiod.go` | Fully-bounded [from, to] interval with Overlaps, OverlapsInclusive, Intersection, Truncate, Equal. | Overlaps returns false for exactly-touching periods ([1,2],[2,3]); use OverlapsInclusive when sequential billing periods must be adjacent. |
| `openperiod.go` | Half/fully-open interval with Intersection, Difference, Union, IsSupersetOf using *time.Time. | Intersection of touching OpenPeriods returns nil; Union with a fully-open ({nil,nil}) period propagates openness. |
| `recurrence.go` | Recurrence (Anchor+Interval) with GetPeriodAt, IterateFromNextAfter/PrevBefore, RecurrenceIterator. | MAX_SAFE_ITERATIONS=1_000_000 caps loops; fractional ISO durations make addIntervalNTimes error; backward iteration past 1733 is unsupported (TODO). |
| `timeline.go` | Generic sorted Timeline[T] of Timed[T] with GetClosedPeriods (n-1 segments) and GetOpenPeriods (n+1 segments). | GetClosedPeriods with one timestamp returns one zero-length period; NewTimeline sorts input so insertion order is irrelevant. |
| `boundedperiod.go` | StartBoundedPeriod with mandatory From and optional *time.Time To; implements Period. | ContainsExclusive treats From as exclusive start — different from ClosedPeriod.ContainsExclusive. |

## Anti-Patterns

- Using time.Duration arithmetic instead of ISODuration for month/year recurrences — wrong period ends.
- Constructing Recurrence{} struct literal without NewRecurrence — skips Validate().
- Treating Overlaps and OverlapsInclusive as interchangeable — sequential billing periods must use Overlaps.
- Constructing OpenPeriod{From:&t, To:&t} expecting a valid point interval — Intersection returns nil for zero-length.

## Decisions

- **Contains uses half-open [from, to) as the default** — Half-open intervals compose without overlap for sequential billing windows, preventing double-counting at the shared boundary.
- **Recurrence iterates from a floating anchor rather than an epoch** — Subscription anchors can be any date; billing periods must align to the original subscription start regardless of when the calculation runs.

## Example: Find the billing period containing a usage timestamp via monthly recurrence

```
import (
    "github.com/openmeterio/openmeter/pkg/timeutil"
    "github.com/openmeterio/openmeter/pkg/datetime"
)

rec, err := timeutil.NewRecurrenceFromISODuration(datetime.DurationMonth, subscriptionStart)
if err != nil { return err }
period, err := rec.GetPeriodAt(usageTime) // period.From <= usageTime < period.To
```

<!-- archie:ai-end -->
