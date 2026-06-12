# service

<!-- archie:ai-start -->

> Implements rating.Service (the pricing engine entrypoint) as a stateless `service struct{}` constructed via New(). It orchestrates the per-line calculation flow: select a pricer for the price type, wrap it with pre/post-calculation mutators in a priceMutator, resolve the billable period, and emit priced rating.DetailedLines with totals. All monetary math routes through currencyx.Calculator.RoundToPrecision.

## Patterns

**Stateless service implementing rating.Service** — The struct holds no fields; all per-call state comes from the StandardLineAccessor input and GenerateDetailedLinesOptions. Construct with New() which returns rating.Service. (`type service struct{}; func New() rating.Service { return &service{} }`)
**Pricer selection by price Type()** — getPricerFor switches on line.GetPrice().Type() (FlatPriceType, UnitPriceType, TieredPriceType, PackagePriceType, DynamicPriceType) to pick a rate.Pricer; unknown types return an error, never a panic or default pricer. (`switch linePrice.Type() { case productcatalog.UnitPriceType: basePricer = rate.Unit{} ... default: return nil, fmt.Errorf("unsupported price type: %s", linePrice.Type()) }`)
**priceMutator pipeline ordering** — GenerateDetailedLines applies PreCalculation mutators to the input, runs the base Pricer, then PostCalculation mutators over the resulting lines. Order is fixed: DiscountUsage (pre) -> Pricer -> DiscountPercentage, MaxAmountCommitment, MinAmountCommitment, Credits (post). Flat prices get only DiscountPercentage + optional Credits and no commitments. (`&priceMutator{PreCalculation: []mutator.PreCalculationMutator{&mutator.DiscountUsage{}}, Pricer: basePricer, PostCalculation: postCalculationMutators}`)
**Options gate mutator inclusion** — rating.GenerateDetailedLinesOptions flags (IgnoreMinimumCommitment, DisableCreditsMutator) conditionally append MinAmountCommitment / Credits to the post-calculation chain. Charges pricing uses IgnoreMinimumCommitment so min-spend only appears after the service period end. (`if !opts.IgnoreMinimumCommitment { postCalculationMutators = append(postCalculationMutators, &mutator.MinAmountCommitment{}) }`)
**Validate input before pricing** — Both GenerateDetailedLines and ResolveBillablePeriod call validateStandardLine / in.Validate() and PricerCalculateInput.Validate() before any calculation; flat lines skip usage extraction, non-flat lines pull metered + pre-line-period quantities. (`if err := validateStandardLine(in); err != nil { return rating.GenerateDetailedLinesResult{}, fmt.Errorf("validating billable line: %w", err) }`)
**Totals computed in service, not pricers** — getTotalsFromDetailedLines walks each DetailedLine, sets per-line Totals via calculateDetailedLineTotals, then sums them (RoundToPrecision) into the UBP line total. CategoryCommitment lines map their amount to ChargesTotal; all others to Amount. (`in.Totals = totals.Sum(lo.Map(in.DetailedLines, func(l rating.DetailedLine, _ int) totals.Totals { return l.Totals })...).RoundToPrecision(calc)`)
**Progressive-billing gated by meter aggregation** — ResolveBillablePeriod force-disables ProgressiveBilling unless the underlying meter aggregation is increase-only (Sum, Count, Max, UniqueCount via isDependingOnIncreaseOnlyMeters). FlatPriceType never allows progressive billing. (`if !meterTypeAllowsProgressiveBilling { in.ProgressiveBilling = false }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the empty `service struct{}` and New() rating.Service constructor. | Do not add fields/dependencies here; the engine is intentionally stateless and driven by inputs. |
| `pricer.go` | getPricerFor maps price type -> rate.Pricer and assembles the priceMutator pre/Pricer/post pipeline; priceMutator.GenerateDetailedLines and ResolveBillablePeriod run the flow. | Mutator ordering is load-bearing (discounts before commitments before credits). New price types must be added to the switch and to FlatPriceType handling, and unknown types must error not default. |
| `detailedline.go` | Public GenerateDetailedLines entrypoint, validateStandardLine, and totals computation (getTotalsFromDetailedLines, calculateDetailedLineTotals). | Do NOT fold discounts/credits into the UBP line totals beyond children — external apps (Stripe) only sync detailed lines, see the in-code WARNING. Flat prices skip Usage population. |
| `billableperiod.go` | ResolveBillablePeriod delegates to the pricer after gating progressive billing on increase-only meter aggregation (isDependingOnIncreaseOnlyMeters). | Non-increase-only meters must be billed in arrears truncated by window; missing feature key or nil meter must error. |
| `options_test.go` | Validates option behavior (WithMinimumCommitmentIgnored, WithCreditsMutatorDisabled) through testutil.RunCalculationTestCase against the real service. | Tests assert full rating.DetailedLines including ChildUniqueReferenceID and Totals; keep new options covered through the harness, not direct mutator calls. |

## Anti-Patterns

- Adding state/dependencies to the `service struct{}` instead of passing them via StandardLineAccessor / GenerateDetailedLinesOptions.
- Reordering or bypassing the priceMutator pipeline (pre -> Pricer -> post) — discounts, commitments, and credits depend on a fixed order.
- Computing totals with float or raw decimal math instead of currencyx.Calculator.RoundToPrecision in calculateDetailedLineTotals / getTotalsFromDetailedLines.
- Injecting discount/credit/commitment logic into totals on the parent UBP line — only detailed-line children are synced externally (see the WARNING comment).
- Returning a default pricer for an unrecognized price Type() instead of erroring; or enabling progressive billing for flat/non-increase-only meters.

## Decisions

- **Pricing engine is a stateless service driven entirely by input accessors and option flags.** — Pricers and mutators must be deterministic and re-runnable (post-calculation mutators are idempotent), so no shared service state is kept across calls.
- **Calculation modeled as a priceMutator wrapping a base Pricer with ordered pre/post mutator slices rather than a monolithic per-price function.** — Lets commitments, discounts, and credits be composed and conditionally toggled per call (charges vs invoice) without duplicating per-price-type logic.
- **Totals are recomputed at the service layer from detailed-line children, not carried from external systems.** — Only detailed lines are synced to external billing apps; parent UBP totals are derived locally and must not embed app-specific discount logic.

## Example: Selecting a pricer and building the mutator pipeline for a non-flat price

```
func getPricerFor(line rating.PriceAccessor, opts rating.GenerateDetailedLinesOptions) (*priceMutator, error) {
	linePrice := line.GetPrice()
	var basePricer rate.Pricer
	switch linePrice.Type() {
	case productcatalog.UnitPriceType:
		basePricer = rate.Unit{}
	case productcatalog.TieredPriceType:
		basePricer = rate.Tiered{}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", linePrice.Type())
	}
	post := []mutator.PostCalculationMutator{&mutator.DiscountPercentage{}, &mutator.MaxAmountCommitment{}}
	if !opts.IgnoreMinimumCommitment {
		post = append(post, &mutator.MinAmountCommitment{})
	}
// ...
```

<!-- archie:ai-end -->
