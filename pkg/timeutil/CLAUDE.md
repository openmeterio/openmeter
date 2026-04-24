# timeutil

<!-- archie:ai-start -->

> Shared time-interval primitives (ClosedPeriod, OpenPeriod, StartBoundedPeriod, Recurrence, Timeline) used throughout billing, subscription, entitlement, and metering to reason about billable windows, recurrence schedules, and event timelines with explicit boundary semantics.

## Patterns

**Three Contains variants with documented boundary semantics** — Every period type implements the Period interface: Contains (inclusive start, exclusive end), ContainsInclusive (both inclusive), ContainsExclusive (both exclusive). Always pick the variant matching the billing/entitlement spec — mixing them silently changes charge window edges. (`period.Contains(t) // [from, to) — default billing window check`)
**Recurrence is anchor-based, not start-based** — Recurrence.Anchor can be any point in time past or future; iteration walks forward/backward from the anchor. Use NewRecurrence or NewRecurrenceFromISODuration constructors — they call Validate(). (`rec, _ := timeutil.NewRecurrence(timeutil.RecurrencePeriodMonth, anchorTime); period, _ := rec.GetPeriodAt(now)`)
**Boundary enum controls inclusive/exclusive edge of iteration** — IterateFromNextAfter and IterateFromPrevBefore accept a Boundary argument. Inclusive returns t itself if it falls on a boundary; Exclusive skips to next/prev. (`next, _ := rec.NextAfter(t, timeutil.Exclusive)`)
**OpenPeriod uses *time.Time to model half-open and unbounded intervals** — nil From means open start; nil To means open end. Operations (Intersection, Difference, Union, IsSupersetOf) handle all nil combinations. Touching periods (sharing an endpoint) do NOT overlap — Intersection returns nil. (`OpenPeriod{From: &t1, To: nil} // half-open, extends to infinity`)
**Timeline auto-sorts and provides segment decomposition** — NewTimeline sorts its input; use GetClosedPeriods() for segment-between-timestamps or GetOpenPeriods() for the fencepost open-interval view (n timestamps → n+1 open periods). (`tl := timeutil.NewSimpleTimeline(times); segs := tl.GetClosedPeriods()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `period.go` | Period interface — three Contains variants all types implement | Any new period type must implement all three methods; missing one will cause compile errors at assertion sites like var _ Period = MyPeriod{} |
| `closedperiod.go` | Fully-bounded [from, to] interval with Overlaps, Intersection, Truncate, Equal helpers | Overlaps returns false for exactly-touching periods ([1,2],[2,3]); use OverlapsInclusive when sequential periods must be treated as adjacent |
| `openperiod.go` | Half or fully-open interval with Intersection, Difference, Union, IsSupersetOf | Intersection of touching open periods returns nil — this is intentional and matches billing window semantics where [t,t] has zero length |
| `recurrence.go` | Recurrence schedule driven by ISODuration interval and anchor; RecurrenceIterator enables bidirectional stepping | MAX_SAFE_ITERATIONS = 1_000_000 cap; fractional ISO durations cause addIntervalNTimes to return an error; backward iteration past 1733 is explicitly not supported per TODO comment |
| `timeline.go` | Generic sorted sequence of Timed[T] with period decomposition and Before/After filters | GetClosedPeriods with a single point returns a zero-length period (From==To); GetOpenPeriods always returns len(times)+1 segments |
| `boundary.go` | Boundary string enum with Validate() | Always call boundaryBehavior.Validate() before using it — Recurrence methods do this internally but callers of custom iterator logic must validate themselves |

## Anti-Patterns

- Using time.Duration arithmetic instead of ISODuration for month/year recurrences — month lengths vary and Duration math gives wrong billing period ends
- Calling Recurrence methods without anchoring to a truncated time — sub-millisecond anchor drift produces off-by-nanosecond period boundaries
- Assuming Overlaps and OverlapsInclusive are interchangeable — sequential billing periods must use Overlaps (returns false) to avoid double-counting
- Constructing OpenPeriod{From: &t, To: &t} expecting a valid point interval — Intersection treats same-start-and-end as zero-length and returns nil

## Decisions

- **Contains uses half-open [from, to) semantics as the default** — Half-open intervals compose without overlap for sequential billing windows: period N ends at T, period N+1 starts at T, Contains(T) is true for N+1 only
- **Recurrence iterates from a floating anchor rather than an epoch** — Subscription anchors can be any date; billing periods must align to the original subscription start regardless of when the calculation runs

## Example: Find the billing period containing a given usage timestamp

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
// period.From <= usageTime < period.To
```

<!-- archie:ai-end -->
