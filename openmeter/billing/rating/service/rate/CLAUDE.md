# rate

<!-- archie:ai-start -->

> The pricing engine's per-price-type strategy layer: each price model (Flat, Unit, Package, Dynamic, VolumeTiered, GraduatedTiered) is a Pricer that turns a billed line + usage into rating.DetailedLines. Constraint: all monetary math goes through alpacadecimal + currencyx.Calculator.RoundToPrecision, never float arithmetic.

## Patterns

**Pricer interface contract** — Every price-type struct implements rate.Pricer (GenerateDetailedLines + ResolveBillablePeriod) and asserts it with a compile-time check. (`type Flat struct{}; var _ Pricer = (*Flat)(nil)`)
**Embed a billable-period mixin** — Metered pricers embed ProgressiveBillingMeteredPricer; non-progressive ones embed NonProgressiveBillingPricer (from base.go) to inherit ResolveBillablePeriod instead of re-implementing it. Flat overrides it inline. (`type Dynamic struct { ProgressiveBillingMeteredPricer }`)
**Convert price via As* before use** — Always extract the concrete price with l.GetPrice().AsFlat()/AsUnit()/AsTiered()/AsDynamic()/AsPackage() and wrap the error with fmt.Errorf; never type-assert the price union directly. (`flatPrice, err := l.GetPrice().AsFlat(); if err != nil { return nil, fmt.Errorf("converting price to flat price: %w", err) }`)
**Stable ChildUniqueReferenceID per line** — Every emitted DetailedLine sets a ChildUniqueReferenceID from rating constants (UsageChildUniqueReferenceID, FlatPriceChildUniqueReferenceID, GraduatedTieredPriceUsageChildUniqueReferenceID fmt'd with tier index) so re-calculation is idempotent and lines can be matched across runs. (`ChildUniqueReferenceID: fmt.Sprintf(rating.GraduatedTieredPriceUsageChildUniqueReferenceID, tierIndex)`)
**Period position gates emission** — Use l.IsFirstInPeriod()/IsLastInPeriod() (from PricerCalculateInput in types.go) to decide whether to emit: in-advance flat bills only first-in-period, in-arrears only last; volume tiered returns nil unless IsLastInPeriod (no progressive support). (`case flatPrice.PaymentTerm == productcatalog.InArrearsPaymentTerm && l.IsLastInPeriod():`)
**Truncate to streaming minimum window** — All period/asOf comparisons in ResolveBillablePeriod truncate to streaming.MinimumWindowSizeDuration so partial-window usage isn't billed prematurely. (`asOf := in.AsOf.Truncate(streaming.MinimumWindowSizeDuration)`)
**Return nil for no-line outcome** — When nothing should be billed (no usage, wrong period position) return (nil, nil) rather than an empty slice; tests treat nil and empty DetailedLines as equivalent. (`if !toBeBilledPackages.IsZero() { return rating.DetailedLines{...}, nil }; return nil, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines Pricer interface, PricerCalculateInput (embeds rating.StandardLineAccessor + CurrencyCalculator + Usage + FullProgressivelyBilledServicePeriod) and its IsFirst/IsLastInPeriod/GetUsage helpers. | GetUsage errors if Usage is nil — callers must ensure usage is populated before calling pricers. |
| `base.go` | ProgressiveBillingMeteredPricer and NonProgressiveBillingPricer ResolveBillablePeriod implementations shared by embedding. | NonProgressiveBillingPricer rejects lines with a SplitLineGroupID via ErrInvoiceProgressiveBillingNotSupported — splitting a non-progressive price is invalid. |
| `tiered.go` | Tiered is a router that dispatches to volume vs graduated based on price.Mode; not a pricer itself beyond delegation. | Unknown price.Mode returns an error — keep the switch exhaustive when adding tier modes. |
| `tieredgraduated.go` | GraduatedTiered plus the TieredPriceCalculator range-splitting algorithm (TierRange, splitTierRangeAtBoundary) used to bill usage across tier boundaries. | The algorithm builds non-overlapping sorted qtyRanges; FromQty is exclusive, ToQty inclusive. Flat-per-tier only billed AtTierBoundary. |
| `tieredvolume.go` | VolumeTiered prices the whole quantity at the single tier it lands in; embeds NonProgressiveBillingPricer. | Returns nil unless IsLastInPeriod — volume tiers have no progressive-billing support. |
| `flat.go` | Flat price; the only pricer that overrides ResolveBillablePeriod inline using GetInvoiceAt vs AsOf. | Defaults empty PaymentTerm to DefaultPaymentTerm and validates it is InAdvance/InArrears, else ValidationError. |
| `package.go` | Package pricing; GetNumberOfPackages ceils qty/packageSize and diffs pre vs post-line package counts to bill only newly-crossed packages. | preLinePeriodPackages is zeroed when IsFirstInPeriod so the first line bills all packages. |
| `dynamic.go` | Dynamic price multiplies usage.Quantity by Multiplier; single usage line in arrears. | Only emits when usage.Quantity.IsPositive(); min-spend handling lives in the mutator layer, not here. |

## Anti-Patterns

- Doing float64 or raw decimal arithmetic instead of routing through l.CurrencyCalculator.RoundToPrecision.
- Type-asserting the price union directly instead of using GetPrice().As*() with a wrapped error.
- Hardcoding ChildUniqueReferenceID strings instead of using rating package constants — breaks line re-matching/idempotency.
- Applying commitments, max-spend, discounts or credits here — those belong to the mutator layer; pricers only produce raw priced lines.
- Emitting lines without checking IsFirst/IsLastInPeriod, double-billing flat fees across split lines.

## Decisions

- **Period-resolution split into Progressive vs NonProgressive mixins embedded by each pricer.** — Billable-period semantics differ only by progressive-billing support, so shared behavior is composed via embedding rather than duplicated per price type.
- **Graduated tiering implemented as an explicit range-splitting algorithm with AtTierBoundary flags.** — Progressive (split) billing must bill flat-per-tier exactly once and unit prices only for the in-scope sub-range; the explicit ranges make this auditable even though it is not the most efficient algorithm (see code comment).

## Example: Add a new price-type pricer

```
package rate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Unit struct{ ProgressiveBillingMeteredPricer }

var _ Pricer = (*Unit)(nil)

func (p Unit) GenerateDetailedLines(l PricerCalculateInput) (rating.DetailedLines, error) {
	unitPrice, err := l.GetPrice().AsUnit()
// ...
```

<!-- archie:ai-end -->
