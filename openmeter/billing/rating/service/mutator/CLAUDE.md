# mutator

<!-- archie:ai-start -->

> Post- and pre-calculation mutators that adjust priced rating.DetailedLines for commitments (min/max spend), percentage/usage discounts, and credit application. Constraint: PostCalculationMutators must be idempotent over re-runs; the usage discount PreCalculationMutator explicitly cannot be (it mutates the usage-based line directly).

## Patterns

**Two mutator interfaces** — types.go defines PostCalculationMutator (Mutate(input, lines) -> lines) and PreCalculationMutator (Mutate(input) -> input). Each concrete mutator is an empty struct with a compile-time interface assertion. (`type MinAmountCommitment struct{}; var _ PostCalculationMutator = (*MinAmountCommitment)(nil)`)
**Append to line discount/credit slices, don't overwrite** — Mutators append to l.AmountDiscounts / pricerResult[idx].CreditsApplied / StandardLineDiscounts.Usage with a stable ChildUniqueReferenceID, so re-running merges rather than duplicates. (`l.AmountDiscounts = append(l.AmountDiscounts, lineDiscount)`)
**Commitment lines use FullProgressivelyBilledServicePeriod** — Min-spend is only emitted IsLastInPeriod and billed across the whole period: when IsProgressivelyBilled, the emitted line's Period is i.FullProgressivelyBilledServicePeriod, not the split period. (`period := i.GetServicePeriod(); if i.IsProgressivelyBilled() { period = i.FullProgressivelyBilledServicePeriod }`)
**Credit allocation must fully consume** — Credits.Mutate walks positive-total lines applying credit; leftover positive credit returns billing.ErrInvoiceLineCreditsNotConsumedFully (over-allocation is a critical invariant violation). (`if creditValueRemaining.IsPositive() { return pricerResult, billing.ErrInvoiceLineCreditsNotConsumedFully }`)
**Reason carries discount provenance** — Discount line items set Reason via billing.NewDiscountReasonFrom(...) and a rating.RateCardDiscountChildUniqueReferenceID keyed by CorrelationID; usage discounts are removed/re-added via removeRateCardUsageDiscounts to stay in sync. (`Reason: billing.NewDiscountReasonFrom(discount.PercentageDiscount)`)
**Round every monetary mutation** — All amount adjustments pass through i.CurrencyCalculator.RoundToPrecision before being stored on a line. (`minimumSpendAmount := i.CurrencyCalculator.RoundToPrecision(commitments.MinimumAmount.Sub(totalBilledAmount))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | The PostCalculationMutator and PreCalculationMutator interfaces both keyed off rate.PricerCalculateInput. | New mutators must pick the right interface — pre runs on input usage, post runs on already-priced lines. |
| `commitments.go` | MinAmountCommitment (emits a min-spend line when total < minimum, last-in-period only) and MaxAmountCommitment (adds overage discounts via AddDiscountForOverage across lines). | Min-spend is in-arrears, last-in-period only; max-spend accumulates totalBilled including previouslyBilledAmount across all lines in order. |
| `credits.go` | Credits PostCalculationMutator distributes each applied credit across positive-total lines, cloning with the consumed amount. | Skips non-positive total lines; under-consumption returns ErrInvoiceLineCreditsNotConsumedFully. TODO marks this for deprecation once charge line mappers own credit projection. |
| `discountpercentage.go` | DiscountPercentage applies a rate-card percentage to each line's TotalAmount; requires a non-empty CorrelationID. | Validates percentage in [0,100]; empty CorrelationID errors. Uses slicesx.MapWithErr. |
| `discountusage.go` | DiscountUsage PreCalculationMutator + exported ApplyUsageDiscount (shared by standard billing and charge line projection) that subtracts discounted quantity from usage and records UsageLineDiscountManaged. | Explicitly non-idempotent — it mutates the usage-based line; removeRateCardUsageDiscounts must clear prior rate-card usage discounts each run to avoid drift. |

## Anti-Patterns

- Making the usage-discount mutator stateful/assuming idempotency — it rewrites the usage line and must re-sync discounts every run.
- Computing discounts/credits with float math instead of CurrencyCalculator.RoundToPrecision.
- Emitting a min-spend line on a non-last split line, or for the split period instead of FullProgressivelyBilledServicePeriod.
- Silently dropping leftover credit instead of returning ErrInvoiceLineCreditsNotConsumedFully.
- Overwriting AmountDiscounts/CreditsApplied/StandardLineDiscounts slices instead of appending with a stable ChildUniqueReferenceID.

## Decisions

- **Pre vs Post calculation mutator split.** — Usage discounts must alter the quantity fed into pricers (pre), while commitments, percentage discounts and credits act on already-priced detailed lines (post).
- **ApplyUsageDiscount exported as a standalone function with explicit input/result structs.** — The usage-discount contract is shared between standard billing and charge line projection, so it lives outside the mutator method to be reused without the PricerCalculateInput wrapper.

## Example: Implement a post-calculation commitment mutator

```
package mutator

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type MinAmountCommitment struct{}

// ...
```

<!-- archie:ai-end -->
