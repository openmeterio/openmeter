# service

<!-- archie:ai-start -->

> Sole stateless implementation of rating.Service: GenerateDetailedLines dispatches a billable line to the correct per-price-type Pricer (in rate/) wrapped by an ordered pre/post mutator pipeline (in mutator/), then aggregates currency-rounded totals. The service struct holds no fields; all state flows through method arguments.

## Patterns

**getPricerFor() is the only pricer/mutator assembly point** — Never instantiate rate.Flat{}/rate.Unit{}/rate.Tiered{} etc. directly; getPricerFor(line, opts) selects the base Pricer and builds the ordered PreCalculation/PostCalculation mutator slices by price type and options. (`linePricer, err := getPricerFor(in, generateOpts)`)
**No Usage for FlatPriceType lines** — input.Usage stays nil for productcatalog.FlatPriceType; non-flat lines must populate it from GetMeteredQuantity()/GetMeteredPreLinePeriodQuantity() before calling the pricer, or PricerCalculateInput.Validate() errors. (`if in.GetPrice().Type() != productcatalog.FlatPriceType { input.Usage = &rating.Usage{...} }`)
**Totals aggregated only in getTotalsFromDetailedLines (service layer)** — Pricers and mutators stay currency-agnostic; after priceMutator.GenerateDetailedLines, the service calls getTotalsFromDetailedLines(out, currencyCalc) which sums per-line totals via totals.Sum and is the only aggregate-level rounding point. (`outWithTotals := getTotalsFromDetailedLines(out, currencyCalc)`)
**currencyx.Calculator for all rounding** — Resolve the calculator once via in.GetCurrency().Calculator() and thread it through PricerCalculateInput; use calc.RoundToPrecision() for every amount — never assign raw alpacadecimal arithmetic to a total. (`amount := calc.RoundToPrecision(line.PerUnitAmount.Mul(line.Quantity))`)
**validateStandardLine before any pricer call** — GenerateDetailedLines and ResolveBillablePeriod call validateStandardLine(in) first: line non-nil, price non-nil, and a non-progressively-billed line's full service period equals its progressively-billed service period. (`if err := validateStandardLine(in); err != nil { return ..., fmt.Errorf("validating billable line: %w", err) }`)
**ProgressiveBilling gated on monotonic meter aggregation** — ResolveBillablePeriod forces in.ProgressiveBilling=false unless isDependingOnIncreaseOnlyMeters returns true (only Sum/Count/Max/UniqueCount); other aggregations (Avg, Latest) bill in arrears. (`if !meterTypeAllowsProgressiveBilling { in.ProgressiveBilling = false }`)
**Option flags gate specific mutators in getPricerFor** — opts.IgnoreMinimumCommitment suppresses MinAmountCommitment (used by charges/partial runs); opts.DisableCreditsMutator suppresses Credits; MaxAmountCommitment is always appended for non-flat lines. (`if !opts.IgnoreMinimumCommitment { postCalculationMutators = append(postCalculationMutators, &mutator.MinAmountCommitment{}) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Zero-field service struct and New() — the sole DI entry point for rating.Service. | service has no fields; adding any breaks the stateless design and risks shared state across concurrent billing runs. |
| `pricer.go` | getPricerFor() maps price types to rate.Pricer + ordered mutators; priceMutator orchestrates pre/post pipeline and delegates ResolveBillablePeriod to the inner Pricer. | A new price type needs a new switch case; missing case returns an unsupported-price-type error at runtime. Ensure DisableCreditsMutator/IgnoreMinimumCommitment propagate. |
| `detailedline.go` | GenerateDetailedLines entry: validates, resolves calculator, builds PricerCalculateInput, runs priceMutator, aggregates via getTotalsFromDetailedLines. | Do not set input.Usage for FlatPriceType; never skip getTotalsFromDetailedLines; CategoryCommitment lines accumulate in ChargesTotal, not Amount. |
| `billableperiod.go` | ResolveBillablePeriod gates ProgressiveBilling via isDependingOnIncreaseOnlyMeters before delegating to the matched pricer. | Returning true for a non-monotonic aggregation (Avg, Latest) causes incorrect progressive billing — only Sum/Count/Max/UniqueCount qualify. |
| `options_test.go` | Demonstrates testutil.RunCalculationTestCase for option-level tests (IgnoreMinimumCommitment, DisableCreditsMutator). | All pricer/mutator tests go through testutil.RunCalculationTestCase; do not instantiate service.New() directly in tests. |

## Anti-Patterns

- Instantiating rate.Unit{}/rate.Flat{}/rate.Tiered{} directly instead of going through getPricerFor()
- Setting input.Usage for FlatPriceType lines — PricerCalculateInput.Validate() errors
- Computing DetailedLine totals inside a pricer or mutator instead of getTotalsFromDetailedLines
- Adding fields/state to the service struct — it is intentionally stateless
- Raw alpacadecimal arithmetic for final amounts without currencyx.Calculator.RoundToPrecision

## Decisions

- **priceMutator wraps a single Pricer with ordered pre/post mutator slices instead of embedding mutation in each Pricer** — Keeps pricing concerns (tier math, period gating) separate from cross-cutting concerns (discounts, commitments, credits) so each evolves and tests in isolation.
- **service struct has no fields; New() returns an interface over an empty struct** — Rating is purely computational with no pools/caches/config; statelessness simplifies testing and avoids shared state across concurrent billing runs.
- **Totals aggregated at the service layer, not inside priceMutator** — Aggregation depends on the currency calculator resolved at the service layer; pricers/mutators stay currency-agnostic and independently testable.

## Example: Adding a new price type: register in getPricerFor() and implement rate.Pricer

```
// pricer.go
case productcatalog.MyNewPriceType:
    basePricer = rate.MyNew{}

// openmeter/billing/rating/service/rate/mynew.go
package rate
import (
    "github.com/openmeterio/openmeter/openmeter/billing/rating"
    "github.com/openmeterio/openmeter/pkg/timeutil"
)
var _ Pricer = MyNew{}
type MyNew struct{ NonProgressiveBillingPricer }
```

<!-- archie:ai-end -->
