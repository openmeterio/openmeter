# datetime

<!-- archie:ai-start -->

> Billing-safe datetime and ISO 8601 duration library. DateTime wraps time.Time with RFC 9557 timezone-annotated formatting/parsing; ISODuration wraps rickb777/period with no-overflow month/year arithmetic and DivisibleBy for subscription cadence validation.

## Patterns

**datetime.Parse for all datetime inputs** — Use datetime.Parse(s) to accept RFC3339/ISO8601/RFC9557 (tz-annotated) strings; never time.Parse directly for user-supplied timestamps. (`dt, err := datetime.Parse("2024-06-30T15:39:00Z")`)
**ISODurationString.Parse() for duration inputs** — Accept durations as ISODurationString and call .Parse(); use ParsePtrOrNil for optional fields. (`d, err := datetime.ISODurationString("P1M").Parse()`)
**DateTime.Add(ISODuration) for calendar arithmetic** — Use DateTime.Add / ISODuration.AddTo — they handle month-end clamping (Jan 31 + 1M = Feb 28) and DST. Never time.Time.Add fixed durations for calendar periods. (`end := startDT.Add(duration)`)
**ISODuration.DivisibleBy for cadence validation** — Validate that a billing period is an integer multiple of a rate-card cadence; tests multiple days-in-month/hours-in-day to be correct under DST. (`ok, err := billingPeriod.DivisibleBy(ratecardCadence)`)
**Pre-defined DurationXxx vars** — Use package-level constants (DurationDay/Month/Year, etc.) from interval.go instead of constructing ISODuration from scratch. (`datetime.DurationMonth`)
**RFC9557Layout for timezone-preserving output** — Use dt.Format(RFC9557Layout) for tz-annotated output; falls back to ISO8601Layout when no named timezone is set. (`dt.Format(datetime.RFC9557Layout)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `datetime.go` | DateTime struct, Add/AddYearsNoOverflow/AddMonthsNoOverflow/AddWeeks/AddDays, shiftClockTo. | shiftClockTo computes wallTimeDiff then d.Time.Add to avoid monotonic clock issues; don't bypass with time.AddDate for month/year. |
| `duration.go` | ISODuration wrapping period.Period: DivisibleBy, Mul, Add, Subtract, AddTo (delegates to DateTime.Add). | DivisibleBy calls Simplify(true) and tests four daysInMonth x three hoursInDay combos; a false negative means the divisor is ambiguous under calendar variation. |
| `parse.go` | Parse tries multiple layouts; RFC9557 [tz] suffix stripped and location loaded via time.LoadLocation. | Relies on strings.LastIndexByte + HasSuffix — multiple bracket pairs or brackets not at end error out. |
| `format.go` | DateTime.Format replaces layoutTZName placeholder with Location().String(). | Unnamed Location falls back to ISO8601; don't assume RFC9557 output for UTC-offset-only times. |
| `durationstring.go` | ISODurationString typed string, Parse() and ParsePtrOrNil(). | Parse wraps period.Parse errors in NewDurationParseError. |
| `interval.go` | Package-level ISODuration constants: DurationSecond..DurationYear. | These are vars, not consts — never assign to them. |
| `testutils.go` | MustParseDateTime/Duration/LoadLocation/ParseTimeInLocation test helpers, in package datetime. | Lives in package datetime (not datetime_test) — accessible from internal tests without import. |

## Anti-Patterns

- Using time.Time.Add(30*24*time.Hour) for 'one month' — use DateTime.Add(DurationMonth).
- Calling time.Parse directly for user-supplied timestamps — go through datetime.Parse.
- Comparing ISODuration with == — use ISODuration.Equal which normalises string form.
- Constructing period.Period directly instead of ISODuration — bypasses no-overflow arithmetic.
- Calling DateTime.Format without RFC9557Layout when Location is named — drops the IANA zone annotation.

## Decisions

- **Custom shiftClockTo for date arithmetic instead of time.AddDate** — time.AddDate can shift the wall clock in ways that break DST transitions; shiftClockTo preserves the monotonic/wall relationship.
- **DivisibleBy tests multiple calendar scenarios rather than exact arithmetic** — ISO calendar divisions are ambiguous; testing across [28,29,30,31] days/month and [23,24,25] hours/day gives a conservative correct answer.

## Example: Parse, advance by ISO duration, format with timezone

```
import "github.com/openmeterio/openmeter/pkg/datetime"

start, err := datetime.Parse("2025-01-31T00:00:00Z")
if err != nil { return err }
end := start.Add(datetime.DurationMonth) // 2025-02-28T00:00:00Z
formatted := end.Format(datetime.RFC9557Layout)
```

<!-- archie:ai-end -->
