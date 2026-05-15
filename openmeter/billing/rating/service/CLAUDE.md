# service

<!-- archie:ai-start -->

> Stateless rating service that dispatches to per-price-type Pricer implementations via a priceMutator pipeline. Orchestrates pre/post-calculation mutators around concrete pricers (rate.Flat, rate.Unit, rate.Tiered, rate.Package, rate.Dynamic) to produce DetailedLines with aggregated totals. This is the sole implementation of rating.Service injected by Wire.

## Patterns

**getPricerFor() is the only entry point to pricer construction** — New code must not instantiate rate.Flat{}, rate.Unit{}, rate.Tiered{}, etc. directly. getPricerFor() selects the correct pricer and assembles the ordered pre/post mutator slices based on price type and GenerateDetailedLinesOptions. (`linePricer, err := getPricerFor(in, generateOpts)`)
**FlatPriceType exclusion from Usage population** — input.Usage must not be set when line price type is productcatalog.FlatPriceType. Non-flat lines must call GetMeteredQuantity() and GetMeteredPreLinePeriodQuantity() to populate input.Usage before passing to linePricer.GenerateDetailedLines. (`if in.GetPrice().Type() != productcatalog.FlatPriceType { input.Usage = &rating.Usage{...} }`)
**getTotalsFromDetailedLines aggregation after pricer pipeline** — Totals must never be computed inside a pricer or mutator. After priceMutator.GenerateDetailedLines returns, call getTotalsFromDetailedLines(out, currencyCalc) which computes per-line totals and sums them via totals.Sum. This is the only place rounding via currencyCalc occurs at the aggregate level. (`outWithTotals := getTotalsFromDetailedLines(out, currencyCalc)`)
**currencyx.Calculator for all rounding** — Obtain the calculator via in.GetCurrency().Calculator() at the service layer and pass it through PricerCalculateInput. Use calc.RoundToPrecision() for all rounding — never do raw alpacadecimal arithmetic and assign to a total without rounding. (`amount := calc.RoundToPrecision(line.PerUnitAmount.Mul(line.Quantity))`)
**opts.IgnoreMinimumCommitment gates MinAmountCommitment mutator** — MinAmountCommitment is appended to postCalculationMutators only when GenerateDetailedLinesOptions.IgnoreMinimumCommitment is false. Charges pricing sets this flag to suppress minimum commitment on partial billing runs. MaxAmountCommitment is always appended for non-flat lines. (`if !opts.IgnoreMinimumCommitment { postCalculationMutators = append(postCalculationMutators, &mutator.MinAmountCommitment{}) }`)
**validateStandardLine before any pricer call** — GenerateDetailedLines calls validateStandardLine(in) first. It enforces: line non-nil, price non-nil, and that non-progressively-billed lines have a full service period equal to their progressively billed service period. (`if err := validateStandardLine(in); err != nil { return ..., fmt.Errorf("validating billable line: %w", err) }`)
**isDependingOnIncreaseOnlyMeters gates ProgressiveBilling in ResolveBillablePeriod** — ResolveBillablePeriod disables in.ProgressiveBilling unless the underlying meter aggregation is Sum, Count, Max, or UniqueCount. Other aggregations (Avg, Latest) must be billed in arrears. The pricer receives the corrected flag. (`if !meterTypeAllowsProgressiveBilling { in.ProgressiveBilling = false }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Zero-field service struct and New() constructor; sole DI entry point for obtaining rating.Service. | service has no fields — all state flows via method arguments. Adding fields here without a compelling reason breaks the stateless design. |
| `pricer.go` | getPricerFor() maps productcatalog price types to concrete rate.Pricer + ordered mutator slices. priceMutator orchestrates the pre/post pipeline and delegates ResolveBillablePeriod to the inner Pricer. | New price types require a new case in the switch; omitting a case returns an unsupported-price-type error at runtime. DisableCreditsMutator and IgnoreMinimumCommitment flags are checked here — ensure they propagate correctly. |
| `detailedline.go` | GenerateDetailedLines entry point: validates input, resolves currency calculator, builds PricerCalculateInput, calls priceMutator, aggregates totals via getTotalsFromDetailedLines. | Do not set input.Usage for FlatPriceType lines. getTotalsFromDetailedLines must always be called on pricer output — never skip it. CategoryCommitment lines accumulate in ChargesTotal, not Amount. |
| `billableperiod.go` | ResolveBillablePeriod delegates to the matched pricer after gating ProgressiveBilling on meter aggregation type via isDependingOnIncreaseOnlyMeters. | Returning true from isDependingOnIncreaseOnlyMeters for a non-monotonic aggregation (e.g. Avg, Latest) causes incorrect progressive billing — only Sum, Count, Max, UniqueCount are monotonic. |
| `options_test.go` | Demonstrates correct usage of testutil.RunCalculationTestCase for options-level tests (IgnoreMinimumCommitment, DisableCreditsMutator). | All pricer/mutator tests must go through testutil.RunCalculationTestCase; do not instantiate service.New() directly in test files. |

## Anti-Patterns

- Instantiating rate.Unit{}/rate.Flat{}/rate.Tiered{} directly in service methods instead of going through getPricerFor()
- Setting input.Usage for FlatPriceType lines — causes PricerCalculateInput.Validate() to return an error
- Computing DetailedLine totals inside a pricer or mutator instead of delegating to getTotalsFromDetailedLines
- Adding state (fields) to the service struct — the service is intentionally stateless; all computation must flow through method parameters
- Using raw alpacadecimal arithmetic for final amounts without rounding via currencyx.Calculator.RoundToPrecision

## Decisions

- **priceMutator wraps a single Pricer with ordered pre/post mutator slices rather than embedding mutation logic inside each Pricer.** — Keeps pricing concerns (tier math, period gating) separate from cross-cutting concerns (discounts, commitments, credits) so each can evolve independently and be tested in isolation.
- **service struct has no fields; New() returns an interface backed by an empty struct.** — The rating service is purely computational — it holds no connection pools, caches, or configuration. Statelessness simplifies testing and avoids accidental shared state across concurrent billing runs.
- **getTotalsFromDetailedLines is called in GenerateDetailedLines (service layer), not inside priceMutator.GenerateDetailedLines.** — Totals aggregation depends on the currency calculator resolved at the service layer; pricers and mutators operate on raw DetailedLines to stay currency-agnostic and independently testable.

## Example: Adding a new price type: register in getPricerFor() and implement rate.Pricer in the rate sub-package

```
// In pricer.go, add a new case to the basePricer switch:
case productcatalog.MyNewPriceType:
    basePricer = rate.MyNew{}

// In openmeter/billing/rating/service/rate/mynew.go:
package rate

import (
    "github.com/openmeterio/openmeter/openmeter/billing/rating"
    "github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ Pricer = MyNew{}

type MyNew struct{ NonProgressiveBillingPricer }
// ...
```

<!-- archie:ai-end -->
