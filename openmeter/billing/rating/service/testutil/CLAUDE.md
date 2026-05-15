# testutil

<!-- archie:ai-start -->

> Test harness for pricer and mutator unit tests. Provides CalculationTestCase, line-mode factories (single-period, mid-period split, last-in-period), and RunCalculationTestCase which wires a real service.New() and compares JSON-serialised DetailedLines output.

## Patterns

**RunCalculationTestCase as the single test entry point** — All pricer/mutator tests call testutil.RunCalculationTestCase(t, CalculationTestCase{...}) — never instantiate service.New() directly in test files. (`testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{Price: ..., LineMode: testutil.SinglePerPeriodLineMode, Usage: ..., Expect: ...})`)
**TestLineMode enum for split-line scenarios** — Use the TestLineMode constants (SinglePerPeriodLineMode, MidPeriodSplitLineMode, LastInPeriodSplitLineMode) to control whether the test line has a SplitLineGroupID and how its period is positioned within TestFullPeriod. (`LineMode: testutil.LastInPeriodSplitLineMode`)
**JSON equality assertion for DetailedLines** — RunCalculationTestCase uses require.JSONEq on marshalled slices to avoid nil-vs-empty-slice false failures and to give readable diffs. Do not use require.Equal on DetailedLines slices directly. (`require.JSONEq(t, string(expectJSON), string(resJSON))`)
**PreviousBilledAmount for progressive billing tests** — Set CalculationTestCase.PreviousBilledAmount to simulate already-billed amounts in previous split lines; the harness inserts a fake prior line with that Totals.Amount. (`PreviousBilledAmount: alpacadecimal.NewFromFloat(90)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ubptest.go` | Single file: defines CalculationTestCase, FeatureUsageResponse, TestLineMode, TestFullPeriod constant, and RunCalculationTestCase. Imports service.New() to run a real pipeline. | The harness always sets line name to 'feature'; expected DetailedLine.Name values in tests must match that prefix (e.g. 'feature: usage in period', 'feature: minimum spend'). |

## Anti-Patterns

- Instantiating service.New() directly in *_test.go files — use RunCalculationTestCase instead
- Constructing billing.StandardLine manually in test files outside this package — use the harness to avoid missing SplitLineHierarchy setup
- Using require.Equal for DetailedLines slices — use RunCalculationTestCase which normalises nil/empty via JSON equality
- Adding test helper state to ubptest.go beyond CalculationTestCase fields — tests should use CalculationTestCase.Options for pricer-level overrides

## Decisions

- **JSON equality rather than deep-equal for DetailedLines** — DetailedLines contains alpacadecimal.Decimal fields whose zero values may differ between nil and empty; JSON normalises these and produces legible diffs in test output.

<!-- archie:ai-end -->
