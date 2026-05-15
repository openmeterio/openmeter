# testutils

<!-- archie:ai-start -->

> Test utility package providing shared fixtures, type converters, and intent constructors for usage-based rating engine tests (delta, periodpreserving, subtract). Converts internal usagebased.DetailedLine and totals.Totals types to float64-based ExpectedDetailedLine / ExpectedTotals structs for readable require.Equal assertions.

## Patterns

**ExpectedDetailedLine for assertion** — Use ToExpectedDetailedLinesWithServicePeriod to convert engine output to []ExpectedDetailedLine before passing to require.Equal. All decimal values are converted to InexactFloat64() so assertions are readable without decimal constructor calls. (`require.Equal(t, []ratingtestutils.ExpectedDetailedLine{{ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID, PerUnitAmount: 10, Quantity: 5, Totals: ratingtestutils.ExpectedTotals{Amount: 50, Total: 50}}}, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))`)
**FormatDetailedLineChildUniqueReferenceID for period-stamped IDs** — Use FormatDetailedLineChildUniqueReferenceID(id, period) to construct the expected period-stamped ChildUniqueReferenceID strings in period-preserving engine tests. Format is '<id>@[<from RFC3339>..<to RFC3339>]'. (`ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1)`)
**NewIntentForTest for standard intent construction** — Use NewIntentForTest(t, servicePeriod, price, discounts) to construct a valid usagebased.Intent for tests. It sets canonical test values (customer-1, USD, feature-1, CreditThenInvoiceSettlementMode) and calls require.NoError(intent.Validate()). (`intent := ratingtestutils.NewIntentForTest(t, servicePeriod, *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(10)}), productcatalog.Discounts{})`)
**NewUnitPriceIntentForTest shortcut** — Use NewUnitPriceIntentForTest(t, servicePeriod, amount) when the test only needs a unit price intent — it calls NewIntentForTest with empty discounts. (`intent := ratingtestutils.NewUnitPriceIntentForTest(t, servicePeriod, alpacadecimal.NewFromInt(3))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testutils.go` | All test utilities are in this single file. Exports ExpectedDetailedLine, ExpectedTotals, ToExpectedDetailedLinesWithServicePeriod, ToExpectedTotals, FormatDetailedLineChildUniqueReferenceID, NewIntentForTest, NewUnitPriceIntentForTest. | ToExpectedDetailedLinesWithServicePeriod always includes ServicePeriod in the expected struct. If your engine output has nil ServicePeriod, use a custom mapping instead of this helper. CorrectsRunID is also included — period-preserving tests must set it on expected lines for corrections. |

## Anti-Patterns

- Do not import app/common from this test utility package — test dependencies must be built from underlying package constructors to avoid import cycles.
- Do not use InexactFloat64() comparisons for currency-precision values outside tests — float64 representation can differ from alpacadecimal rounding; these helpers are test-only.
- Do not add production code to this package — it is exclusively for test fixtures and converters.
- Do not construct usagebased.Intent manually in engine tests — always use NewIntentForTest or NewUnitPriceIntentForTest to ensure the intent passes Validate().

## Decisions

- **All decimal values are converted to InexactFloat64() in expected structs rather than using alpacadecimal.Decimal in assertions.** — Float64 values make test case literals readable without alpacadecimal.NewFromFloat(...) calls throughout test files. Engine tests operate on currency-scale amounts where float64 precision is sufficient for assertion purposes.

<!-- archie:ai-end -->
