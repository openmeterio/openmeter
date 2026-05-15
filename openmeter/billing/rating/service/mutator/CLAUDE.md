# mutator

<!-- archie:ai-start -->

> Post- and pre-calculation mutation pipeline that transforms rating.DetailedLines after (or PricerCalculateInput before) a pricer runs. Each mutator applies one orthogonal concern — min/max spend commitments, credits allocation, percentage discounts, usage-quantity discounts — keeping pricing logic composable and isolated.

## Patterns

**Interface segregation by phase** — Implement PostCalculationMutator (Mutate(PricerCalculateInput, DetailedLines) (DetailedLines, error)) for post-calculation changes, or PreCalculationMutator (Mutate(PricerCalculateInput) (PricerCalculateInput, error)) for pre-calculation input changes. Never conflate the two. (`var _ PostCalculationMutator = (*MinAmountCommitment)(nil)`)
**Compile-time interface assertion** — Every mutator struct must have a blank-identifier assertion against its interface immediately after the type declaration. (`var _ PostCalculationMutator = (*Credits)(nil)`)
**ChildUniqueReferenceID for idempotency** — Every DetailedLine appended by a mutator must set ChildUniqueReferenceID using constants from the rating package (rating.MinSpendChildUniqueReferenceID, rating.RateCardDiscountChildUniqueReferenceID) to allow idempotent re-calculation. (`ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID`)
**IsLastInPeriod / IsFirstInPeriod guards** — Commitment and discount mutators that apply only once per billing period must guard on i.IsLastInPeriod() or i.IsFirstInPeriod() to avoid double-billing in progressive billing split-line scenarios. (`if !i.IsLastInPeriod() { return pricerResult, nil }`)
**CurrencyCalculator for rounding** — All monetary arithmetic must use i.CurrencyCalculator.RoundToPrecision(...) rather than raw alpacadecimal operations to respect the invoice currency precision. (`minimumSpendAmount := i.CurrencyCalculator.RoundToPrecision(commitments.MinimumAmount.Sub(totalBilledAmount))`)
**ErrInvoiceLineCreditsNotConsumedFully sentinel** — If credits are not fully consumed after iterating all lines, return billing.ErrInvoiceLineCreditsNotConsumedFully — never silently discard the remaining credit value. (`return pricerResult, billing.ErrInvoiceLineCreditsNotConsumedFully`)
**MergeDiscountsByChildUniqueReferenceID for usage discount idempotency** — DiscountUsage (PreCalculationMutator) always calls removeRateCardUsageDiscounts first to strip stale entries, then rewrites via MergeDiscountsByChildUniqueReferenceID so reruns stay idempotent. (`out.StandardLineDiscounts.Usage = out.StandardLineDiscounts.Usage.MergeDiscountsByChildUniqueReferenceID(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines the two mutator interfaces (PostCalculationMutator, PreCalculationMutator). All other files implement one of these. | Do not add methods to either interface without updating every implementor. |
| `commitments.go` | MinAmountCommitment and MaxAmountCommitment post-mutators. MinAmount appends a commitment DetailedLine; MaxAmount iterates lines and calls AddDiscountForOverage. | MinAmountCommitment reads i.GetPreviouslyBilledAmount() — the period coverage for the new line must use i.FullProgressivelyBilledServicePeriod when progressively billed. |
| `credits.go` | Allocates pre-computed CreditsApplied across DetailedLines in order, appending CreditApplied entries per line. | Credits must be fully consumed; any positive remainder triggers ErrInvoiceLineCreditsNotConsumedFully. Skips non-positive TotalAmount lines. |
| `discountpercentage.go` | Applies a rate-card percentage discount to every line's TotalAmount as an AmountLineDiscountManaged entry. | ChildUniqueReferenceID is built from rating.RateCardDiscountChildUniqueReferenceID format string — CorrelationID on the discount must be non-empty or the function returns an error. |
| `discountusage.go` | Pre-mutator that reduces usage quantity based on rate-card usage discount. Not idempotent on the raw line — always strips old usage discounts before reapplying via ApplyUsageDiscount. | ApplyUsageDiscount manipulates PricerCalculateInput.Usage.Quantity directly; PreLinePeriodQuantity must be adjusted for already-consumed discount from earlier split lines. |

## Anti-Patterns

- Adding state to mutator structs — all mutators are stateless value types; all context flows through PricerCalculateInput
- Skipping ChildUniqueReferenceID on appended DetailedLines — breaks idempotent recalculation
- Doing currency rounding with raw alpacadecimal arithmetic instead of CurrencyCalculator.RoundToPrecision
- Applying min/max commitments outside of IsLastInPeriod guard — causes double-billing on progressive billing
- Implementing both PreCalculationMutator and PostCalculationMutator in a single struct — conflates the two phases

## Decisions

- **Two separate interfaces (Pre vs Post) rather than a single pipeline step** — Pre-mutators modify usage quantity before pricing (changing what gets priced), post-mutators modify resulting DetailedLines. Conflating them would force every mutator to understand both the pricer input and output shapes.
- **Stateless value-type mutators with no constructors** — All needed context is in PricerCalculateInput; stateless types allow safe reuse across concurrent calculations and require no DI wiring in app/common.

## Example: Adding a new PostCalculationMutator that caps line amounts

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
