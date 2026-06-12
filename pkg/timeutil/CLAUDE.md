# timeutil

<!-- archie:ai-start -->

> Value-type time-interval and recurrence algebra used across billing, subscription, entitlement and credit domains. Provides Period implementations (ClosedPeriod, OpenPeriod, StartBoundedPeriod), Recurrence iteration, and Timeline period derivation — all immutable, allocation-light, dependency-free except pkg/datetime.

## Patterns

**Period interface contract: three containment semantics** — Every period type implements Period (period.go): Contains (inclusive start, exclusive end), ContainsInclusive (both ends), ContainsExclusive (neither end). New period types must satisfy all three with a compile-time `var _ Period = T{}` assertion. (`var _ Period = ClosedPeriod{}  // closedperiod.go; same in openperiod.go, boundedperiod.go`)
**Value receivers, no mutation** — All methods take value receivers and return new structs or pointers; periods are copied freely. Intersection/Union/Difference return fresh values (or nil for OpenPeriod.Intersection meaning 'no overlap'). (`func (p ClosedPeriod) Intersection(other ClosedPeriod) *ClosedPeriod  // returns nil when !newFrom.Before(newTo)`)
**Boundary enum gates recurrence direction** — Recurrence iteration helpers take a Boundary (Inclusive/Exclusive, boundary.go) and call Boundary.Validate() first. Inclusive returns t when it matches an anchor point; Exclusive steps to the next/prev value. (`func (r Recurrence) NextAfter(t time.Time, boundaryBehavior Boundary) (time.Time, error)`)
**Iteration cap via MAX_SAFE_ITERATIONS** — All recurrence stepping loops (iterateFromNextAfterInclusive, iterateFromPrevBeforeInclusive) guard against runaway iteration with MAX_SAFE_ITERATIONS (1_000_000) and return an error rather than looping forever. (`if ic >= MAX_SAFE_ITERATIONS { return RecurrenceIterator{}, fmt.Errorf("recurrence.NextAfter: too many iterations") }`)
**Calendar-correct interval addition via datetime.ISODuration** — RecurrenceInterval embeds datetime.ISODuration; addIntervalNTimes uses Interval.Mul(n) then ISODuration.AddTo, returning an error when the duration is fractional/non-exact. Handles variable-length months without overflow. (`n, ok := interval.AddTo(t); if !ok { return ..., fmt.Errorf("next recurrence calculation wasn't exact, likely a fractional duration") }`)
**Generic Timeline derives periods from sorted timestamps** — Timeline[T] sorts a clone of inputs on construction (NewTimeline) and exposes GetClosedPeriods / GetOpenPeriods. Wrap values with AsTimed(fn) so the timeline knows how to extract each element's time. Use SimpleTimeline/NewSimpleTimeline for plain time.Time. (`tl := timeutil.NewSimpleTimeline(times); periods := tl.GetOpenPeriods()`)
**Validate() collects errors, never panics** — Validate methods return descriptive errors; Recurrence.Validate aggregates via errors.Join over a []error slice. Period validators reject from.After(to) and zero anchors. (`var errs []error; ...; return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `period.go` | Defines the Period interface (ContainsInclusive/ContainsExclusive/Contains). | Adding a period type without all three methods breaks the `var _ Period` assertions. |
| `closedperiod.go` | Bounded [from,to] period with Overlaps, Intersection, ContainsPeriodInclusive, Truncate, IsEmpty. | Overlaps treats exactly-sequential periods ([1,2],[2,3]) as NOT overlapping; OverlapsInclusive treats them as overlapping. Intersection returns nil for zero-length or touching periods. |
| `openperiod.go` | Nullable-bound period (From/To *time.Time) with Intersection, Union, Difference, IsSupersetOf, Closed(). | nil bound means open/unbounded. Intersection returns nil for no overlap and &OpenPeriod{} for both-empty. Union of an empty period returns empty. Closed() errors if either bound is nil. |
| `boundedperiod.go` | StartBoundedPeriod (required From, optional *To) — used where start is mandatory but end may be open. | Open() lifts it to OpenPeriod; no Intersection/Union here. |
| `recurrence.go` | Recurrence{Interval,Anchor} with NextAfter/PrevBefore/GetPeriodAt and the RecurrenceIterator (Next/Prev). | Anchor is arbitrary, not necessarily first occurrence. GetPeriodAt yields a period where Contains(t) holds (inclusive start, exclusive end). Construct via NewRecurrence/NewRecurrenceFromISODuration to get validation. |
| `timeline.go` | Generic Timeline[T] + Timed[T] wrapper; derives ClosedPeriods/OpenPeriods between sorted timestamps. | GetOpenPeriods on a single time returns TWO periods (open-start→t and t→open-end). NewTimeline clones+sorts; original slice untouched. |
| `boundary.go` | Boundary string enum (Inclusive/Exclusive) with Validate. | Only valid for recurrence boundary behavior; passing an arbitrary string fails Validate. |
| `compare.go` | Compare(a,b) returning int(a.Sub(b)). | Returns nanosecond delta as int, not a normalized -1/0/1; only the sign is meaningful and large gaps risk int overflow on 32-bit. |

## Anti-Patterns

- Mutating a period in place or assuming a method has a pointer receiver — all are value receivers returning new values.
- Treating OpenPeriod.Intersection==nil as 'empty intersection' — nil means NO overlap, &OpenPeriod{} means fully-open.
- Hand-iterating recurrence with raw time.Add for months/years — use Recurrence/RecurrenceInterval so variable-length months and overflow are handled.
- Confusing Overlaps vs OverlapsInclusive (and Contains vs ContainsInclusive) at sequential boundaries — pick the variant matching the boundary semantics you need.
- Constructing Recurrence{} as a literal and skipping Validate — use NewRecurrence so positive interval and non-zero anchor are enforced.

## Decisions

- **Period types are immutable value structs implementing a shared interface.** — Time intervals are passed by value across billing/subscription/credit hot paths; immutability avoids aliasing bugs and the interface lets callers stay generic over closed/open/start-bounded forms.
- **Recurrence stepping is bounded by MAX_SAFE_ITERATIONS and returns errors.** — AGENTS forbids panics in non-test paths; a misconfigured fractional interval or far-past anchor could otherwise loop indefinitely.
- **Interval math delegates to datetime.ISODuration.AddTo with an exactness check.** — Calendar arithmetic (month/year boundaries, leap days) is non-trivial; centralizing in datetime keeps recurrence correct and signals fractional durations as errors instead of silent drift.

## Example: Iterate a monthly recurrence and get the containing period for a timestamp

```
import (
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

rec, err := timeutil.NewRecurrence(timeutil.RecurrencePeriodMonth, anchor)
if err != nil {
	return err
}
period, err := rec.GetPeriodAt(t) // ClosedPeriod where period.Contains(t) is true
if err != nil {
	return err
}
next, err := rec.NextAfter(t, timeutil.Exclusive)
```

<!-- archie:ai-end -->
