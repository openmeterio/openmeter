# testutils

<!-- archie:ai-start -->

> Shared test fixtures and assertion adapters for the usage-based rating engines (delta, periodpreserving, subtract). Provides float64-projected expectation types and intent constructors so rating tests assert on simple values instead of alpacadecimal/Totals internals.

## Patterns

**Expectation types are float64 projections of domain values** — ExpectedDetailedLine and ExpectedTotals mirror usagebased.DetailedLine / totals.Totals but with float64 fields. ToExpectedDetailedLinesWithServicePeriod and ToExpectedTotals convert via InexactFloat64() so tests compare floats, not decimals. (`require.Equal(t, expected, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))`)
**Intent constructors validate and t.Helper** — NewIntentForTest builds a usagebased.Intent (chargesmeta.Intent + price/discounts), calls require.NoError(t, intent.Validate()), and is t.Helper(). NewUnitPriceIntentForTest wraps it for the common unit-price case. (`intent := ratingtestutils.NewIntentForTest(t, fullServicePeriod, tc.price, tc.discounts)`)
**FormatDetailedLineChildUniqueReferenceID encodes the period suffix** — Produces '<id>@[<from RFC3339>..<to RFC3339>]' — the canonical period-stamped child reference used by periodpreserving expectations. Tests must use this helper, not hand-built strings, to stay in sync with the engine's stamping. (`ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testutils.go` | All exported helpers: ExpectedDetailedLine, ExpectedTotals, ToExpectedDetailedLinesWithServicePeriod, ToExpectedTotals, FormatDetailedLineChildUniqueReferenceID, NewIntentForTest, NewUnitPriceIntentForTest. | The default fixture currency is USD, SettlementMode is CreditThenInvoiceSettlementMode, ManagedBy is billing.SubscriptionManagedLine; FullServicePeriod/BillingPeriod default to the same servicePeriod. The package is imported as ratingtestutils by delta, periodpreserving, and subtract tests — keep it test-only and free of production dependencies. |

## Anti-Patterns

- Hand-writing period-stamped child reference strings instead of FormatDetailedLineChildUniqueReferenceID.
- Comparing alpacadecimal/Totals directly instead of projecting to float64 via the ToExpected* helpers.
- Adding production (non-test) behavior or non-test imports to this testutils package.

## Decisions

- **Project decimals to float64 for assertions.** — Per AGENTS.md, prefer require.Equal on InexactFloat64() with plain float literals over decimal equality assertions where precision allows — keeps rating tests readable.

## Example: Build a validated intent and assert engine output as floats

```
intent := ratingtestutils.NewUnitPriceIntentForTest(t, servicePeriod, alpacadecimal.NewFromInt(3))
out, _ := New(billingratingservice.New()).Rate(t.Context(), Input{Intent: intent, CurrentPeriod: ...})
require.Equal(t, expectedLines, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))
require.Equal(t, expectedTotals, ratingtestutils.ToExpectedTotals(out.DetailedLines.SumTotals()))
```

<!-- archie:ai-end -->
