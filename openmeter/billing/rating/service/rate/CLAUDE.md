# rate

<!-- archie:ai-start -->

> Concrete Pricer implementations — one per productcatalog price type (Flat, Unit, Dynamic, Package, Tiered/Graduated, Tiered/Volume) — plus shared billable-period resolution. Each pricer converts a PricerCalculateInput into rating.DetailedLines and decides whether a line is billable at a given time.

## Patterns

**Pricer interface with compile-time assertion** — Every pricer satisfies Pricer (GenerateDetailedLines + ResolveBillablePeriod) and declares var _ Pricer = (*MyPricer)(nil) after the type. (`var _ Pricer = (*Unit)(nil)`)
**Embed base pricers for period resolution** — Usage-based pricers supporting progressive billing embed ProgressiveBillingMeteredPricer; pricers that never split embed NonProgressiveBillingPricer. Do not reimplement ResolveBillablePeriod. (`type Dynamic struct { ProgressiveBillingMeteredPricer }`)
**ChildUniqueReferenceID constants from rating package** — All DetailedLine entries set ChildUniqueReferenceID using rating package constants, never raw string literals. (`ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID`)
**IsLastInPeriod / IsFirstInPeriod payment-term gating** — InAdvance lines emit only when l.IsFirstInPeriod(); InArrears lines only when l.IsLastInPeriod() (or always for per-usage lines), preventing duplicate billing across split lines. (`case flatPrice.PaymentTerm == productcatalog.InAdvancePaymentTerm && l.IsFirstInPeriod():`)
**TieredPriceCalculator callback for graduated tier math** — Graduated tiered math uses GraduatedTiered.TieredPriceCalculator with a TierCallbackFn receiving TierCallbackInput per tier boundary — do not reimplement range-splitting inline. (`p.TieredPriceCalculator(TieredPriceCalculatorInput{TieredPrice: price, FromQty: usage.PreLinePeriodQuantity, ToQty: ..., TierCallbackFn: func(in TierCallbackInput) error {...}})`)
**billing.ValidationError for invariant violations** — Return billing.ValidationError{Err: ...} (not fmt.Errorf) when a configuration violates a billing invariant such as progressive billing on a flat-price line. (`return nil, billing.ValidationError{Err: billing.ErrInvoiceProgressiveBillingNotSupported}`)
**Tiered struct as mode router** — Tiered inspects price.Mode and delegates to volume or graduated sub-pricer; new modes must be added to both switch blocks (GenerateDetailedLines and ResolveBillablePeriod). (`case productcatalog.GraduatedTieredPrice: return p.graduated.GenerateDetailedLines(l)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Pricer interface and PricerCalculateInput (embeds StandardLineAccessor + CurrencyCalculator + Usage + StandardLineDiscounts); IsLastInPeriod/IsFirstInPeriod helpers. | Usage is a pointer; call GetUsage() for a safe copy — direct dereference panics when Usage is nil. |
| `base.go` | ProgressiveBillingMeteredPricer and NonProgressiveBillingPricer providing shared ResolveBillablePeriod; progressive truncates asOf to streaming.MinimumWindowSizeDuration. | NonProgressiveBillingPricer rejects lines with a SplitLineGroupID; embedding it blocks progressive billing for that price type. |
| `tieredgraduated.go` | GraduatedTiered and TieredPriceCalculator; splitTierRangeAtBoundary handles FromQty/ToQty cutting across tier boundaries for progressive billing. | TieredPriceCalculatorInput.Validate() must pass; FromQty and ToQty non-negative and FromQty <= ToQty. |
| `tiered.go` | Router dispatching to volume/graduated by price.Mode — the only external entry point for tiered pricing. | A new tiered mode requires updates to both switch blocks (GenerateDetailedLines and ResolveBillablePeriod). |
| `flat.go` | FlatPrice pricer with custom ResolveBillablePeriod using line.InvoiceAt instead of period end; no progressive billing. | Compares invoiceAt vs asOf (both truncated); InAdvance emits only on IsFirstInPeriod, InArrears only on IsLastInPeriod. |

## Anti-Patterns

- Reimplementing tier-boundary range splitting outside TieredPriceCalculator
- Returning fmt.Errorf for billing invariant violations — use billing.ValidationError{Err: ...}
- Setting ChildUniqueReferenceID as a raw string literal instead of rating constants
- Adding progressive-billing support to a pricer embedding NonProgressiveBillingPricer without switching the base type
- Calling PricerCalculateInput.Usage directly instead of GetUsage() — panics when Usage is nil

## Decisions

- **Separate struct per price type rather than a single switch-heavy function** — Each price type has distinct period-resolution semantics and line structure; separate structs allow independent testing and embedding the correct base pricer.
- **TieredPriceCalculator as a reusable callback-driven algorithm** — Graduated tiered pricing needs range splitting across FromQty/ToQty boundaries; centralising with TierCallbackFn avoids duplicating the range logic in both volume and graduated pricers.

## Example: Add a new usage-based pricer for a 'stepped' price type

```
package rate

import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Stepped struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Stepped)(nil)

func (p Stepped) GenerateDetailedLines(l PricerCalculateInput) (rating.DetailedLines, error) {
	usage, err := l.GetUsage()
// ...
```

<!-- archie:ai-end -->
