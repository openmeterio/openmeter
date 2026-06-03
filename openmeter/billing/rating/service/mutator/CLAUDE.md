# mutator

<!-- archie:ai-start -->

> Pre- and post-calculation mutation pipeline that transforms rating.DetailedLines (or PricerCalculateInput) around a pricer run. Each mutator applies one orthogonal pricing concern — min/max spend commitments, credits allocation, percentage discounts, usage-quantity discounts — keeping pricing logic composable and isolated.

## Patterns

**Interface segregation by phase** — Implement PostCalculationMutator (Mutate(PricerCalculateInput, DetailedLines)->DetailedLines) for output changes, or PreCalculationMutator (Mutate(PricerCalculateInput)->PricerCalculateInput) for input changes. Never conflate the two in one struct. (`var _ PostCalculationMutator = (*MinAmountCommitment)(nil)`)
**Compile-time interface assertion** — Every mutator struct declares a blank-identifier assertion against its interface immediately after the type declaration. (`var _ PostCalculationMutator = (*Credits)(nil)`)
**ChildUniqueReferenceID for idempotency** — Every DetailedLine appended by a mutator sets ChildUniqueReferenceID using rating package constants (rating.MinSpendChildUniqueReferenceID, rating.RateCardDiscountChildUniqueReferenceID) so recalculation is idempotent. (`ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID`)
**IsLastInPeriod / IsFirstInPeriod guards** — Commitment/discount mutators that apply once per billing period guard on i.IsLastInPeriod()/i.IsFirstInPeriod() to avoid double-billing across progressive split lines. (`if !i.IsLastInPeriod() { return pricerResult, nil }`)
**CurrencyCalculator for rounding** — All monetary arithmetic uses i.CurrencyCalculator.RoundToPrecision(...) rather than raw alpacadecimal ops, respecting invoice currency precision. (`minimumSpendAmount := i.CurrencyCalculator.RoundToPrecision(commitments.MinimumAmount.Sub(totalBilledAmount))`)
**ErrInvoiceLineCreditsNotConsumedFully sentinel** — If credits are not fully consumed after iterating all lines, return billing.ErrInvoiceLineCreditsNotConsumedFully — never silently discard the remaining credit value. (`return pricerResult, billing.ErrInvoiceLineCreditsNotConsumedFully`)
**MergeDiscountsByChildUniqueReferenceID for usage discount idempotency** — DiscountUsage (pre-mutator) calls removeRateCardUsageDiscounts first to strip stale entries, then rewrites via MergeDiscountsByChildUniqueReferenceID so reruns stay idempotent. (`out.StandardLineDiscounts.Usage = out.StandardLineDiscounts.Usage.MergeDiscountsByChildUniqueReferenceID(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines the two mutator interfaces (PostCalculationMutator, PreCalculationMutator); all other files implement one. | Do not add methods to either interface without updating every implementor. |
| `commitments.go` | MinAmountCommitment and MaxAmountCommitment post-mutators; MinAmount appends a commitment DetailedLine, MaxAmount calls AddDiscountForOverage per line. | MinAmountCommitment reads i.GetPreviouslyBilledAmount(); the new line's period must use i.FullProgressivelyBilledServicePeriod when progressively billed. |
| `credits.go` | Allocates pre-computed CreditsApplied across DetailedLines in order, appending CreditApplied per line; skips non-positive TotalAmount lines. | Credits must be fully consumed; any positive remainder triggers ErrInvoiceLineCreditsNotConsumedFully. |
| `discountpercentage.go` | Applies a rate-card percentage discount to every line's TotalAmount as an AmountLineDiscountManaged entry. | ChildUniqueReferenceID built from rating.RateCardDiscountChildUniqueReferenceID; CorrelationID must be non-empty or the function errors. |
| `discountusage.go` | Pre-mutator reducing usage quantity per rate-card usage discount via ApplyUsageDiscount. | Not idempotent on the raw line — strips old usage discounts before reapplying; PreLinePeriodQuantity must subtract discount already consumed by earlier split lines. |

## Anti-Patterns

- Adding state to mutator structs — all are stateless value types; context flows through PricerCalculateInput
- Skipping ChildUniqueReferenceID on appended DetailedLines — breaks idempotent recalculation
- Currency rounding with raw alpacadecimal instead of CurrencyCalculator.RoundToPrecision
- Applying min/max commitments outside the IsLastInPeriod guard — double-bills on progressive billing
- Implementing both PreCalculationMutator and PostCalculationMutator in one struct

## Decisions

- **Two separate interfaces (Pre vs Post) rather than a single pipeline step** — Pre-mutators change what gets priced (usage quantity); post-mutators change resulting DetailedLines. Conflating forces every mutator to understand both input and output shapes.
- **Stateless value-type mutators with no constructors** — All context is in PricerCalculateInput; stateless types are safe under concurrent calculation and need no DI wiring in app/common.

## Example: Add a new PostCalculationMutator that caps line amounts

```
package mutator

import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

type MyCap struct{}

var _ PostCalculationMutator = (*MyCap)(nil)

func (m *MyCap) Mutate(i rate.PricerCalculateInput, lines rating.DetailedLines) (rating.DetailedLines, error) {
	if !i.IsLastInPeriod() {
		return lines, nil
	}
// ...
```

<!-- archie:ai-end -->
