# service

<!-- archie:ai-start -->

> Stateless rating service that implements rating.Service by dispatching to per-price-type Pricer implementations via a priceMutator pipeline. Orchestrates pre/post-calculation mutators around concrete pricers to produce DetailedLines with totals.

## Patterns

**priceMutator pipeline assembly** — getPricerFor() constructs a priceMutator combining a concrete rate.Pricer with ordered PreCalculation and PostCalculation mutator slices. Caller must not bypass getPricerFor() and instantiate pricers directly. (`priceMutator{PreCalculation: []mutator.PreCalculationMutator{&mutator.DiscountUsage{}}, Pricer: rate.Unit{}, PostCalculation: [...]}`)
**opts.IgnoreMinimumCommitment gate** — MinAmountCommitment mutator is only appended when GenerateDetailedLinesOptions.IgnoreMinimumCommitment is false. Charges pricing sets this flag to suppress minimum commitment on partial billing runs. (`if !opts.IgnoreMinimumCommitment { postCalculationMutators = append(postCalculationMutators, &mutator.MinAmountCommitment{}) }`)
**Usage nil check before FlatPrice** — input.Usage is only populated for non-FlatPrice lines. Flat lines must not set Usage; non-flat lines must call GetMeteredQuantity/GetMeteredPreLinePeriodQuantity and set input.Usage before calling linePricer.GenerateDetailedLines. (`if in.GetPrice().Type() != productcatalog.FlatPriceType { input.Usage = &rating.Usage{...} }`)
**CurrencyCalculator for all rounding** — currencyx.Calculator obtained from in.GetCurrency().Calculator() must be used for all arithmetic rounding via calc.RoundToPrecision. Raw alpacadecimal.Mul results are only rounded by the calculator. (`amount := calc.RoundToPrecision(line.PerUnitAmount.Mul(line.Quantity))`)
**getTotalsFromDetailedLines aggregation** — After linePricer.GenerateDetailedLines returns, getTotalsFromDetailedLines computes per-DetailedLine totals and sums them with totals.Sum. Do not compute totals inline in the pricer; always go through this function. (`outWithTotals := getTotalsFromDetailedLines(out, currencyCalc)`)
**validateStandardLine before any pricer call** — GenerateDetailedLines calls validateStandardLine first to enforce that price is non-nil and that progressively-billed service period matches service period for non-progressive lines. (`if err := validateStandardLine(in); err != nil { return ..., fmt.Errorf("validating billable line: %w", err) }`)
**isDependingOnIncreaseOnlyMeters for progressive billing gate** — ResolveBillablePeriod forcibly disables ProgressiveBilling unless the underlying meter aggregation is Sum, Count, Max, or UniqueCount. Other aggregations must be billed in arrears. (`if !meterTypeAllowsProgressiveBilling { in.ProgressiveBilling = false }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Zero-field service struct and New() constructor; sole entry point for DI consumers to obtain rating.Service. | service has no fields — all state is passed via method arguments; do not add fields without reason. |
| `pricer.go` | getPricerFor() factory that maps productcatalog price types to concrete rate.Pricer + mutator slices; priceMutator orchestrates the pipeline. | New price types require a new case in the switch; forgetting a case returns an unsupported-price-type error at runtime. |
| `detailedline.go` | GenerateDetailedLines entry point: validates input, resolves currency calculator, builds PricerCalculateInput, calls pipeline, aggregates totals. | Usage must not be set for FlatPriceType lines; getTotalsFromDetailedLines must always be called on pricer output. |
| `billableperiod.go` | ResolveBillablePeriod delegates to the pricer after gating ProgressiveBilling on meter aggregation type. | Returning true from isDependingOnIncreaseOnlyMeters for a non-monotonic aggregation (e.g. Avg, Latest) causes incorrect progressive billing. |

## Anti-Patterns

- Instantiating rate.Unit{}/rate.Flat{}/etc. directly in service methods instead of going through getPricerFor()
- Setting input.Usage for FlatPriceType lines — causes validation errors in PricerCalculateInput.Validate()
- Computing DetailedLine totals inside a pricer or mutator instead of delegating to getTotalsFromDetailedLines
- Adding state (fields) to the service struct — the service is intentionally stateless; all computation inputs must flow through method parameters
- Using raw alpacadecimal arithmetic for final amounts without passing through currencyx.Calculator.RoundToPrecision

## Decisions

- **priceMutator wraps a single Pricer with ordered pre/post mutator slices rather than embedding mutation logic inside each Pricer.** — Keeps pricing concerns (tier math, period gating) separate from cross-cutting concerns (discounts, commitments, credits) so each can evolve independently.
- **service struct has no fields; New() returns an interface backed by an empty struct.** — The rating service is purely computational — it holds no connection pools, caches, or configuration. Statelessness simplifies testing and avoids accidental shared state across concurrent billing runs.
- **getTotalsFromDetailedLines is called in GenerateDetailedLines, not inside priceMutator.GenerateDetailedLines.** — Totals aggregation depends on the currency calculator which is resolved at the service layer; pricers and mutators operate on raw DetailedLines to stay currency-agnostic.

## Example: Adding a new price type pricer: register it in getPricerFor and implement rate.Pricer

```
// In pricer.go, add a new case:
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
