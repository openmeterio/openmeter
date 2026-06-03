# testutils

<!-- archie:ai-start -->

> Test-only package providing shared fixtures, type converters, and intent constructors for usage-based rating engine tests (delta, periodpreserving, subtract). Converts internal usagebased.DetailedLine / totals.Totals to float64-based ExpectedDetailedLine / ExpectedTotals for readable require.Equal assertions.

## Patterns

**ExpectedDetailedLine for assertion** — Convert engine output via ToExpectedDetailedLinesWithServicePeriod before require.Equal; decimals become InexactFloat64() so assertions read without decimal constructors. (`require.Equal(t, []ratingtestutils.ExpectedDetailedLine{{PerUnitAmount: 10, Quantity: 5, Totals: ratingtestutils.ExpectedTotals{Amount: 50, Total: 50}}}, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))`)
**FormatDetailedLineChildUniqueReferenceID for period-stamped IDs** — Build expected period-stamped child reference strings ('<id>@[<from RFC3339>..<to RFC3339>]') for period-preserving tests. (`ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1)`)
**NewIntentForTest for standard intent construction** — Construct a valid usagebased.Intent with canonical values (customer-1, USD, feature-1, CreditThenInvoiceSettlementMode) and require.NoError(intent.Validate()). (`intent := ratingtestutils.NewIntentForTest(t, servicePeriod, *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(10)}), productcatalog.Discounts{})`)
**NewUnitPriceIntentForTest shortcut** — Use for unit-price-only intents; calls NewIntentForTest with empty discounts. (`intent := ratingtestutils.NewUnitPriceIntentForTest(t, servicePeriod, alpacadecimal.NewFromInt(3))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testutils.go` | All utilities in one file: ExpectedDetailedLine, ExpectedTotals, ToExpectedDetailedLinesWithServicePeriod, ToExpectedTotals, FormatDetailedLineChildUniqueReferenceID, NewIntentForTest, NewUnitPriceIntentForTest. | ToExpectedDetailedLinesWithServicePeriod always includes ServicePeriod and CorrectsRunID; period-preserving tests must set CorrectsRunID on expected correction lines. |

## Anti-Patterns

- Importing app/common — test dependencies must be built from underlying constructors to avoid import cycles.
- Using InexactFloat64() comparisons for currency-precision values outside tests — float64 can differ from alpacadecimal rounding; these helpers are test-only.
- Adding production code to this package — it is exclusively test fixtures and converters.
- Constructing usagebased.Intent manually in engine tests — use NewIntentForTest / NewUnitPriceIntentForTest so the intent passes Validate().

## Decisions

- **All decimal values are converted to InexactFloat64() in expected structs rather than using alpacadecimal.Decimal in assertions.** — Float64 literals make test cases readable; engine tests operate at currency scale where float64 precision suffices for assertions.

<!-- archie:ai-end -->
