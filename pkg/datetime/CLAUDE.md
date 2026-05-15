# datetime

<!-- archie:ai-start -->

> Billing-safe datetime and ISO 8601 duration library. DateTime wraps time.Time with RFC 9557 timezone-annotated formatting/parsing; ISODuration wraps rickb777/period with no-overflow month/year arithmetic and DivisibleBy for subscription cadence validation.

## Patterns

**Parse for all datetime inputs** — Use datetime.Parse(s) to accept RFC3339, ISO8601, and RFC9557 (timezone-annotated) strings. Never use time.Parse directly for user-supplied timestamps. (`dt, err := datetime.Parse("2024-06-30T15:39:00Z")`)
**ISODurationString.Parse() for duration inputs** — Accept durations as ISODurationString and call .Parse() to obtain ISODuration. Use ParsePtrOrNil for optional fields. (`d, err := datetime.ISODurationString("P1M").Parse()`)
**DateTime.Add(ISODuration) for calendar arithmetic** — Always use DateTime.Add or ISODuration.AddTo for date arithmetic — these handle month-end clamping (Jan 31 + 1M = Feb 28) and DST. Never use time.Time.Add with fixed durations for calendar periods. (`end := startDT.Add(duration)`)
**ISODuration.DivisibleBy for cadence validation** — Used to validate that a billing period is an integer multiple of a rate card cadence. Tests multiple days-in-month/hours-in-day scenarios for correctness under DST. (`ok, err := billingPeriod.DivisibleBy(ratecardCadence)`)
**Pre-defined DurationXxx vars** — Use the package-level constants (DurationDay, DurationMonth, DurationYear, etc.) from interval.go instead of constructing ISODuration from scratch for common periods. (`datetime.DurationMonth`)
**DateTime.Format with RFC9557Layout for timezone-preserving output** — Use dt.Format(RFC9557Layout) to produce timezone-annotated output. Falls back to ISO8601Layout when no named timezone is set. (`dt.Format(datetime.RFC9557Layout)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `datetime.go` | DateTime struct, Add/AddYearsNoOverflow/AddMonthsNoOverflow/AddWeeks/AddDays, shiftClockTo for wall-clock-safe arithmetic. | shiftClockTo computes wallTimeDiff then calls d.Time.Add to avoid re-triggering monotonic clock issues. Do not bypass with time.AddDate for month/year arithmetic. |
| `duration.go` | ISODuration wrapping period.Period, DivisibleBy, Mul, Add, Subtract, AddTo (delegates to DateTime.Add). | DivisibleBy calls Simplify(true) and tests four daysInMonth x three hoursInDays combinations — a false negative means the divisor is ambiguous under calendar variation. |
| `parse.go` | Parse tries multiple layouts; RFC9557 [tz] suffix is stripped and location loaded via time.LoadLocation. | Relies on strings.LastIndexByte + HasSuffix — multiple bracket pairs or brackets not at end return an error. |
| `format.go` | DateTime.Format replaces layoutTZName placeholder with the actual Location().String(). | If Location is unnamed (empty string), falls back to ISO8601 layout; do not assume RFC9557 output for UTC-offset-only times. |
| `durationstring.go` | ISODurationString typed string, Parse() and ParsePtrOrNil(). | Parse wraps period.Parse errors in NewDurationParseError for consistent error messaging. |
| `interval.go` | Package-level ISODuration constants: DurationSecond, DurationMinute, DurationHour, DurationDay, DurationWeek, DurationMonth, DurationYear. | These are vars, not consts — they can be accidentally mutated. Never assign to them. |
| `testutils.go` | MustParseDateTime, MustParseDuration, MustLoadLocation, MustParseTimeInLocation — test helpers only, live in the main package (not testutils sub-package). | testutils.go is in package datetime not package datetime_test — accessible from internal tests without import. |

## Anti-Patterns

- Using time.Time.Add(30 * 24 * time.Hour) for 'one month' — use DateTime.Add(DurationMonth) to handle month-end clamping
- Calling time.Parse directly for user-supplied timestamps — always go through datetime.Parse for RFC9557 support
- Comparing ISODuration values with == — use ISODuration.Equal(*ISODuration) which normalises string form
- Constructing period.Period directly instead of ISODuration — bypasses no-overflow arithmetic in AddTo
- Calling DateTime.Format without providing a RFC9557Layout when the Location has a named timezone — produces output without the IANA zone annotation

## Decisions

- **Custom shiftClockTo for all date arithmetic instead of time.AddDate** — time.AddDate can shift the wall clock in ways that break DST transitions; shiftClockTo preserves the monotonic/wall relationship by computing the diff and calling Add.
- **DivisibleBy tests multiple calendar scenarios rather than exact arithmetic** — ISO calendar divisions are ambiguous (1 year / hours depends on DST); testing across [28,29,30,31] days/month and [23,24,25] hours/day provides a conservative correct answer.

## Example: Parse, advance by ISO duration, format with timezone

```
import "github.com/openmeterio/openmeter/pkg/datetime"

start, err := datetime.Parse("2025-01-31T00:00:00Z")
if err != nil { return err }
end := start.Add(datetime.DurationMonth) // 2025-02-28T00:00:00Z
formatted := end.Format(datetime.RFC9557Layout)
```

<!-- archie:ai-end -->
