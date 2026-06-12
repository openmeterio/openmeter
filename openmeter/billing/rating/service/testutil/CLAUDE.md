# testutil

<!-- archie:ai-start -->

> Shared table-driven test harness for the rating pricers and mutators. Constraint: tests assert through the real production path (service.New().GenerateDetailedLines) against a hand-built billing.StandardLine fixture, comparing expected vs actual via require.JSONEq.

## Patterns

**CalculationTestCase + RunCalculationTestCase** — Every rate/mutator test builds a CalculationTestCase (Price, Discounts, LineMode, Usage, Expect, ExpectErrorIs, PreviousBilledAmount, CreditsApplied, Options) and calls testutil.RunCalculationTestCase(t, tc). (`testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{ Price: ..., LineMode: testutil.SinglePerPeriodLineMode, Usage: ..., Expect: rating.DetailedLines{...} })`)
**LineMode selects split scenario** — TestLineMode (SinglePerPeriodLineMode / MidPeriodSplitLineMode / LastInPeriodSplitLineMode) configures line.Period and, for split modes, attaches a fake SplitLineGroup + SplitLineHierarchy so first/last-in-period logic is exercised. (`LineMode: testutil.LastInPeriodSplitLineMode`)
**Usage set via both raw and metered fields** — Usage.LinePeriodQty/PreLinePeriodQty are assigned to both Quantity/MeteredQuantity and PreLinePeriodQuantity/MeteredPreLinePeriodQuantity on UsageBased to mirror real snapshots. (`line.UsageBased.Quantity = &tc.Usage.LinePeriodQty; line.UsageBased.MeteredQuantity = &tc.Usage.LinePeriodQty`)
**JSON equality assertion** — Expected and actual DetailedLines are marshaled and compared with require.JSONEq; empty/nil results short-circuit to pass. PreviousBilledAmount is injected via a fake prior line in the SplitLineHierarchy. (`require.JSONEq(t, string(expectJSON), string(resJSON))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ubptest.go` | Sole file: TestLineMode constants, TestFullPeriod (fixed 2021-01-01..02), FeatureUsageResponse, CalculationTestCase, and RunCalculationTestCase harness driving service.New().GenerateDetailedLines. | PreviousBilledAmount is modeled as a fake line in fakeHierarchy with no Period so it is always in scope for NetAmount; ExpectErrorIs uses require.ErrorIs and returns early. |

## Anti-Patterns

- Calling individual pricers/mutators directly in tests instead of going through RunCalculationTestCase / service.New() — bypasses the real engine ordering.
- Asserting on struct equality of decimals instead of relying on the JSONEq comparison the harness performs.
- Setting only Quantity without MeteredQuantity (or only the non-metered field) — both must be set to match production snapshots.

## Decisions

- **Drive every pricer/mutator test through the assembled production service rather than unit-isolating each strategy.** — Pricing correctness depends on engine ordering (pricer then mutators) and split/period state, so the harness exercises the full GenerateDetailedLines path with a realistic StandardLine fixture.

<!-- archie:ai-end -->
