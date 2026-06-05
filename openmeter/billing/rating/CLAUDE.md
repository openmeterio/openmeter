# rating

<!-- archie:ai-start -->

> The pricing-engine domain root for billing. It declares the rating.Service interface (ResolveBillablePeriod + GenerateDetailedLines), the line-accessor contracts the engine reads pricing/usage from, the DetailedLine result model with its monetary math, and the stable ChildUniqueReferenceID constants that name every generated detailed line. The concrete engine lives in the service/ child; this package is the type/contract surface plus pure decimal arithmetic.

## Patterns

**Accessor-interface input contracts** — The engine never takes a concrete line struct; it reads via PriceAccessor / GatheringLineAccessor / StandardLineAccessor interfaces. New pricing inputs must be added as accessor methods, not as fields on a request struct. (`StandardLineAccessor.GetMeteredQuantity() (*alpacadecimal.Decimal, error); GetRateCardDiscounts() billing.Discounts; IsProgressivelyBilled() bool`)
**Stable ChildUniqueReferenceID constants** — Every detailed line a pricer emits is keyed by a const from const.go (e.g. UsageChildUniqueReferenceID, GraduatedTieredPriceUsageChildUniqueReferenceID format string). These IDs are the external sync key — reuse the existing const, never inline a literal. (`VolumeUnitPriceChildUniqueReferenceID = "volume-tiered-price"; GraduatedTieredPriceUsageChildUniqueReferenceID = "graduated-tiered-%d-price-usage"`)
**All money math via currencyx.Calculator.RoundToPrecision** — DetailedLine.TotalAmount and AddDiscountForOverage round every intermediate via in.Currency.RoundToPrecision before comparing or subtracting. Never compare raw decimals or floats directly. (`total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity)); total = total.Sub(in.AmountDiscounts.SumAmount(in.Currency))`)
**Total = perUnit*qty minus discounts minus credits** — TotalAmount(getTotalAmountInput) is the single canonical formula: round(perUnit*qty) - AmountDiscounts.SumAmount - CreditsApplied.SumAmount. DetailedLine.TotalAmount and DetailedLines.Sum both route through it. (`func TotalAmount(in getTotalAmountInput) alpacadecimal.Decimal { total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity)); total = total.Sub(in.AmountDiscounts.SumAmount(in.Currency)); ... }`)
**Functional options for engine behaviour** — GenerateDetailedLines takes ...GenerateDetailedLinesOption (WithMinimumCommitmentIgnored, WithCreditsMutatorDisabled) collapsed via NewGenerateDetailedLinesOptions. Add behaviour flags as new With* options, not as new method overloads. (`func WithCreditsMutatorDisabled() GenerateDetailedLinesOption { return func(o *GenerateDetailedLinesOptions){ o.DisableCreditsMutator = true } }`)
**Validate inputs before pricing** — Input structs (ResolveBillablePeriodInput) and result models (DetailedLine) carry Validate() returning fmt.Errorf on missing/invalid fields. DetailedLine.Validate rejects negative PerUnitAmount but intentionally allows negative Quantity (usage corrections). (`func (i ResolveBillablePeriodInput) Validate() error { if i.Line == nil { return fmt.Errorf("line is required") } ... }`)
**Max-spend overage applied as an AmountLineDiscount** — AddDiscountForOverage models the maximum-spend cap by appending a billing.AmountLineDiscountManaged with ChildUniqueReferenceID=billing.LineMaximumSpendReferenceID and Reason=MaximumSpendDiscount, returning a copy of the line — it never mutates totals directly. (`i.AmountDiscounts = append(i.AmountDiscounts, billing.AmountLineDiscountManaged{ AmountLineDiscount: billing.AmountLineDiscount{ Amount: discountAmount, LineDiscountBase: billing.LineDiscountBase{ ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID), Reason: billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}) }}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Declares the rating.Service interface (ResolveBillablePeriod, GenerateDetailedLines), the option type + With* helpers, and the result/input structs (GenerateDetailedLinesResult with FinalUsage/FinalStandardLineDiscounts/Totals, ResolveBillablePeriodInput with its Validate). | This is interface-only; the implementation is in service/. ResolveBillablePeriodInput.Validate requires Line, FeatureMeters, and non-zero AsOf. |
| `line.go` | Defines the three accessor interfaces the engine consumes: PriceAccessor (base: GetPrice/GetServicePeriod/GetFeatureKey), GatheringLineAccessor (gathering-phase, adds split-group/invoiceAt/ID), StandardLineAccessor (standard-phase, adds currency, metered quantity, credits, discounts, progressive-billing accessors). | Add new pricing inputs here as accessor methods. GetMeteredQuantity / GetMeteredPreLinePeriodQuantity / GetPreviouslyBilledAmount return errors — callers must handle them. |
| `detailedline.go` | DetailedLine result model + pure monetary math: TotalAmount/getTotalAmountInput, DetailedLines.Sum, and AddDiscountForOverage for max-spend capping. Carries AmountDiscounts, CreditsApplied, totals.Totals, Category, PaymentTerm. | TotalAmount is the only money formula — do not duplicate it. AddDiscountForOverage returns a copy (value receiver); reassign the result. Validate allows negative Quantity but not negative PerUnitAmount. |
| `const.go` | Canonical ChildUniqueReferenceID string constants for every detailed-line variant (usage, flat-price, unit-price, volume, graduated-tiered format strings, rate-card-discount). | These are external sync keys; changing a value breaks downstream line reconciliation. Per-type IDs are flagged for deprecation toward generic single-child names — prefer existing consts over new literals. |
| `detailedline_test.go` | Table tests for DetailedLine.Validate (negative qty allowed, negative perUnit rejected) and AddDiscountForOverage covering no-overage, currency rounding, partial discount, and 100% discount cases. | Asserts exact discount Amount and the fixed description format 'Maximum spend discount for charges over <amount>' — keep formatMaximumSpendDiscountDescription in sync. |

## Anti-Patterns

- Adding concrete request fields instead of extending PriceAccessor/GatheringLineAccessor/StandardLineAccessor — the engine reads exclusively through these interfaces.
- Computing line totals with raw alpacadecimal or float math instead of routing through TotalAmount / currencyx.Calculator.RoundToPrecision.
- Inlining a ChildUniqueReferenceID string literal rather than using the const.go constant — these are external sync keys.
- Mutating DetailedLine totals in place; AddDiscountForOverage has a value receiver and returns a new line, so the result must be reassigned.
- Rejecting negative Quantity in DetailedLine.Validate — negative quantity is a supported usage-correction case (only PerUnitAmount must be non-negative).

## Decisions

- **Engine inputs are accessor interfaces, not structs.** — Gathering vs standard invoice phases expose different data; interfaces let the same pricer code read lazily-computed/errorable values (metered quantity, previously-billed amount) without coupling to a concrete line type.
- **Detailed-line monetary math lives in this root package as pure functions.** — Totals must be recomputed deterministically from perUnit*qty minus discounts/credits with consistent currency rounding; centralizing it in TotalAmount keeps the service/ pricers and external callers from diverging.
- **Max-spend cap is expressed as a discount line, not a total override.** — Only detailed-line children are synced externally; encoding the cap as an AmountLineDiscount with MaximumSpendDiscount reason keeps the parent line total derivable and auditable.

## Example: Canonical detailed-line total: round(perUnit*qty) then subtract discounts and credits via the currency calculator.

```
import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TotalAmount(in getTotalAmountInput) alpacadecimal.Decimal {
	total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))
	total = total.Sub(in.AmountDiscounts.SumAmount(in.Currency))
	total = total.Sub(in.CreditsApplied.SumAmount(in.Currency))
	return total
}
```

<!-- archie:ai-end -->
