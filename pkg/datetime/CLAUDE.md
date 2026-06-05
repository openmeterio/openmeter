# datetime

<!-- archie:ai-start -->

> Custom datetime/duration value types layered over the standard library, github.com/rickb777/period, and govalues/decimal. Provides DateTime (RFC9557-aware time.Time wrapper) and ISODuration with calendar-correct, no-overflow date arithmetic used across billing, subscription, and entitlement period math.

## Patterns

**DateTime wraps time.Time, never replaces it** — DateTime embeds time.Time so all stdlib methods pass through; only Format/Parse/Add behavior is overridden. Construct with NewDateTime(t) and unwrap with AsTime(). (`type DateTime struct { time.Time }; func (t DateTime) AsTime() time.Time { return t.Time }`)
**No-overflow calendar arithmetic via shiftClockTo** — AddYearsNoOverflow/AddMonthsNoOverflow clamp to the last valid day of the target month (Jan 31 + 1M = Feb 28), and all Add* helpers route through shiftClockTo to mimic time.Add wall-clock behavior across DST. (`dt.AddYearsNoOverflow(y).AddMonthsNoOverflow(m).AddWeeks(w).AddDays(d)...`)
**ISODuration wraps period.Period** — ISODuration embeds period.Period; build with NewISODuration(y,m,w,d,h,min,s) and round-trip strings through ISODurationString. Arithmetic (Add/Subtract/Mul) returns wrapped errors via NewDurationArithmeticError. (`type ISODuration struct { period.Period }`)
**String types for ISO8601 wire form** — ISODurationString is the serialized form; use .Parse()/.ParsePtrOrNil() to get an ISODuration and .ISOString()/.ISOStringPtrOrNil() to go back. Nil-safe pointer variants exist for optional fields. (`d, err := ISODurationString("P1Y2M").Parse()`)
**Multi-format Parse with RFC9557 bracket timezone** — Parse() accepts RFC3339, ISO8601 (incl. Zulu/fractional), and RFC9557 'ts[Area/City]' forms; it strictly rejects malformed brackets (nested, trailing text, empty tz). DateTime.UnmarshalJSON delegates to Parse. (`Parse("2021-07-01T12:34:56-04:00[America/New_York]")`)
**Errors are constructors in errors.go** — All error values come from New*Error helpers (NewDateTimeParseError, NewDurationParseError, NewDurationArithmeticError, NewInvalidTimezoneError) wrapping the underlying cause with %w; do not inline fmt.Errorf for these cases. (`return NewDurationParseError(string(i), err)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `datetime.go` | DateTime type + Add(ISODuration) and the no-overflow Add* arithmetic helpers | MarshalJSON emits RFC3339 only (drops bracket tz and sub-second precision); shiftClockTo is the load-bearing helper — bypassing it breaks DST correctness |
| `duration.go` | ISODuration type, arithmetic, DivisibleBy, AddTo, convertPeriodToSeconds | DivisibleBy brute-forces 28/29/30/31-day months x 23/24/25-hour days, so 'P1Y divisible by PT8H' is intentionally false; AddTo always returns precise=true (kept for back-compat) |
| `parse.go` | Multi-layout Parse entry point | Strict bracket validation — any text after ']' or empty/nested brackets is an error; timezone is loaded via time.LoadLocation so unknown zones fail |
| `constants.go` | RFC9557/ISO8601 layout strings | layoutTZName is the literal 'Europe/Budapest' placeholder swapped out during Format; do not treat it as a default timezone |
| `format.go` | Format() with RFC9557 timezone-name substitution | Falls back to ISO8601 layouts (fallbackRFC9557FormatLayout) when location string is empty |
| `durationstring.go` | ISODurationString parse/serialize helpers incl. nil-safe pointer variants | Parse() wraps failures in NewDurationParseError, not the raw period error |
| `interval.go` | Predefined DurationSecond..DurationYear constants | Use these instead of re-constructing NewISODuration for unit durations |
| `testutils.go` | Test helpers MustLoadLocation/MustParseDateTime/MustParseDuration | Non-_test file but test-only; takes *testing.T and calls t.Fatalf |

## Anti-Patterns

- Doing calendar math with time.AddDate directly instead of DateTime.Add* — reintroduces month-end overflow bugs the No-overflow helpers exist to prevent
- Constructing time.Time/period.Period values directly and bypassing shiftClockTo, losing DST wall-clock semantics
- Relying on DateTime.MarshalJSON to preserve bracket timezone or nanosecond precision (it serializes RFC3339)
- Inlining fmt.Errorf for parse/arithmetic failures instead of the New*Error constructors
- Treating layoutTZName ('Europe/Budapest') as a real default timezone

## Decisions

- **Wrap rickb777/period and govalues/decimal rather than use time.Duration** — time.Duration cannot represent calendar units (months/years) needed for billing periods, and decimal seconds avoid float drift
- **DivisibleBy tests every realistic month/day length combination** — Captures DST and variable-month-length edge cases so subscription cadence validation is conservative and correct

## Example: Parse an ISO8601 duration and add it to a timestamp with no-overflow calendar math

```
import "github.com/openmeterio/openmeter/pkg/datetime"

d, err := datetime.ISODurationString("P1M").Parse()
if err != nil { return err }
next := datetime.NewDateTime(start).Add(d).AsTime() // 2024-01-31 + P1M = 2024-02-29
```

<!-- archie:ai-end -->
