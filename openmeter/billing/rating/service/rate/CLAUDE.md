# rate

<!-- archie:ai-start -->

> Concrete Pricer implementations — one per productcatalog price type (Flat, Unit, Dynamic, Package, Tiered/Graduated, Tiered/Volume) — plus shared billable-period resolution helpers. Each pricer converts a PricerCalculateInput into rating.DetailedLines and determines whether a line is billable at a given point in time.

## Patterns

**Pricer interface with compile-time assertion** — Every pricer struct must satisfy the Pricer interface (GenerateDetailedLines + ResolveBillablePeriod) and declare var _ Pricer = (*MyPricer)(nil) after the type. (`var _ Pricer = (*Unit)(nil)`)
**Embed base pricers for period resolution** — Usage-based pricers that support progressive billing embed ProgressiveBillingMeteredPricer; pricers that never support split lines embed NonProgressiveBillingPricer. Do not reimplement ResolveBillablePeriod manually. (`type Dynamic struct { ProgressiveBillingMeteredPricer }`)
**ChildUniqueReferenceID constants from rating package** — All DetailedLine entries must set ChildUniqueReferenceID using constants defined in the parent rating package, never raw string literals. (`ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID`)
**IsLastInPeriod / IsFirstInPeriod payment-term gating** — InAdvance lines are emitted only when l.IsFirstInPeriod() is true; InArrears lines only when l.IsLastInPeriod() is true (or always for per-usage lines). This prevents duplicate billing across progressive split lines. (`case flatPrice.PaymentTerm == productcatalog.InAdvancePaymentTerm && l.IsFirstInPeriod():`)
**TieredPriceCalculator callback for graduated tier math** — Graduated tiered calculations must use GraduatedTiered.TieredPriceCalculator with TieredPriceCalculatorInput; the TierCallbackFn receives TierCallbackInput per tier boundary. Do not reimplement range-splitting inline. (`p.TieredPriceCalculator(TieredPriceCalculatorInput{TieredPrice: price, FromQty: usage.PreLinePeriodQuantity, ToQty: ..., TierCallbackFn: func(in TierCallbackInput) error {...}})`)
**billing.ValidationError for invariant violations** — Return billing.ValidationError{Err: ...} (not fmt.Errorf) when a line configuration violates a billing invariant such as progressive billing on a flat-price line. (`return nil, billing.ValidationError{Err: billing.ErrInvoiceProgressiveBillingNotSupported}`)
**Tiered struct as mode router** — The Tiered struct is a router that inspects price.Mode and delegates to its volume or graduated sub-pricer. New tiered modes must be added to both switch blocks in GenerateDetailedLines and ResolveBillablePeriod. (`case productcatalog.GraduatedTieredPrice: return p.graduated.GenerateDetailedLines(l)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines Pricer interface and PricerCalculateInput (embeds rating.StandardLineAccessor + CurrencyCalculator + Usage + StandardLineDiscounts). IsLastInPeriod / IsFirstInPeriod helpers live here. | PricerCalculateInput.Usage is a pointer; always call GetUsage() for a safe copy — direct dereference panics when Usage is nil. |
| `base.go` | ProgressiveBillingMeteredPricer and NonProgressiveBillingPricer providing shared ResolveBillablePeriod logic. Progressive truncates asOf to streaming.MinimumWindowSizeDuration. | NonProgressiveBillingPricer rejects lines with a SplitLineGroupID; embedding it in a new pricer type blocks progressive billing for that price type. |
| `tieredgraduated.go` | GraduatedTiered and TieredPriceCalculator. splitTierRangeAtBoundary handles the edge case where FromQty/ToQty cut across tier boundaries for progressive billing. | TieredPriceCalculatorInput.Validate() must pass before calling TieredPriceCalculator; both FromQty and ToQty must be non-negative and FromQty <= ToQty. |
| `tiered.go` | Router that dispatches to volume/graduated based on price.Mode. Only external entry point for tiered pricing. | Adding a new tiered mode requires updates to both switch blocks (GenerateDetailedLines and ResolveBillablePeriod). |
| `flat.go` | FlatPrice pricer with custom ResolveBillablePeriod using line.InvoiceAt instead of period end. Does not support progressive billing. | Flat prices compare invoiceAt vs asOf (both truncated); InAdvance emits only on IsFirstInPeriod, InArrears only on IsLastInPeriod. |

## Anti-Patterns

- Reimplementing tier-boundary range splitting outside TieredPriceCalculator
- Returning fmt.Errorf for billing invariant violations — use billing.ValidationError{Err: ...}
- Setting ChildUniqueReferenceID as a raw string literal instead of using rating package constants
- Adding progressive-billing support to a pricer that embeds NonProgressiveBillingPricer without switching the base type
- Calling PricerCalculateInput.Usage directly instead of GetUsage() — panics when Usage is nil

## Decisions

- **Separate structs per price type rather than a single switch-heavy function** — Each price type has distinct period-resolution semantics and line structure; separate structs allow independent testing and embedding of the correct base pricer.
- **TieredPriceCalculator as a reusable callback-driven algorithm** — Graduated tiered pricing requires range splitting across FromQty/ToQty boundaries; centralising the algorithm with TierCallbackFn avoids duplicating the range logic in both volume and graduated pricers.

## Example: Adding a new usage-based pricer for a hypothetical 'stepped' price type

```
package rate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Stepped struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Stepped)(nil)

// ...
```

<!-- archie:ai-end -->
