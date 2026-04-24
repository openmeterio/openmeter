# rating

<!-- archie:ai-start -->

> Public types and interfaces for the billing rating layer: defines Service, DetailedLine, StandardLineAccessor, GatheringLineAccessor, and child unique reference ID constants used by the rating/service implementation and all callers (charges, billing worker).

## Patterns

**ChildUniqueReferenceID constants** — All child line identifiers must use the named constants in const.go (e.g. UsageChildUniqueReferenceID, FlatPriceChildUniqueReferenceID). Never hardcode these strings inside pricer or service logic. (`ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID`)
**currencyx.Calculator for all monetary arithmetic** — Every amount computation on DetailedLine must route through currencyx.Calculator.RoundToPrecision before being stored; raw alpacadecimal arithmetic is only acceptable as an intermediate step. (`total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))`)
**GenerateDetailedLinesOption functional options** — Behavioural variations (e.g. IgnoreMinimumCommitment) are passed as variadic GenerateDetailedLinesOption functions; never add boolean flags to the interface signature. (`svc.GenerateDetailedLines(line, rating.WithMinimumCommitmentIgnored())`)
**Accessor interfaces, not concrete types** — Callers supply lines via StandardLineAccessor or GatheringLineAccessor interfaces defined in line.go; the rating package must not import or reference concrete billing line structs directly. (`func (s *service) GenerateDetailedLines(in StandardLineAccessor, opts ...GenerateDetailedLinesOption) (GenerateDetailedLinesResult, error)`)
**AddDiscountForOverage for max-spend capping** — Maximum-spend cap discounts must be applied by calling DetailedLine.AddDiscountForOverage with an AddDiscountInput; do not compute overage discounts inline in pricers. (`line = line.AddDiscountForOverage(rating.AddDiscountInput{BilledAmountBeforeLine: prev, MaxSpend: max, Currency: calc})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `const.go` | Single source of truth for all ChildUniqueReferenceID string values; used by pricers, tests, and callers to label DetailedLine children. | Adding new price types without a corresponding constant here causes silent mismatches in child-line correlation. |
| `detailedline.go` | Defines DetailedLine value type, TotalAmount, AddDiscountForOverage, and DetailedLines.Sum; this is the canonical output unit of the rating service. | TotalAmount already subtracts AmountDiscounts and CreditsApplied — double-subtracting in callers produces incorrect totals. |
| `line.go` | Accessor interfaces (StandardLineAccessor, GatheringLineAccessor, PriceAccessor) that decouple the rating package from concrete billing line types. | Adding methods here forces all implementors to update; prefer adding helpers to service.go instead. |
| `service.go` | Service interface, GenerateDetailedLinesResult, ResolveBillablePeriodInput, and option types. Actual implementation lives in rating/service sub-package. | ResolveBillablePeriodInput.Validate() is the authoritative check — do not duplicate validation in callers. |

## Anti-Patterns

- Hardcoding ChildUniqueReferenceID strings instead of using the constants in const.go
- Adding fields to the service struct in the rating/service implementation — the service is intentionally stateless
- Setting input.Usage for FlatPriceType lines — the pricer validates this and returns an error
- Computing DetailedLine totals (rounding, discount subtraction) outside of TotalAmount / getTotalsFromDetailedLines
- Importing concrete billing line types directly in this package — use the accessor interfaces in line.go

## Decisions

- **Service interface lives in rating package root; implementation in rating/service sub-package** — Callers (charges, billing worker) import only the interface and types without pulling in implementation dependencies.
- **DetailedLine is a value type with no pointer receivers except AddDiscountForOverage which returns a new value** — Immutable-by-convention pipeline: pricers produce new DetailedLine values rather than mutating shared state, making mutation order explicit.
- **Behavioural knobs use functional options (GenerateDetailedLinesOption) rather than input struct fields** — Keeps the primary input path (StandardLineAccessor) stable while allowing opt-in overrides (IgnoreMinimumCommitment) without breaking callers.

## Example: Computing a line's final total with discount and credit deduction

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

calc, _ := currencyx.Code(currency.USD).Calculator()
line := rating.DetailedLine{
	Name:                   "Unit usage",
	ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
	Quantity:               alpacadecimal.NewFromFloat(10),
	PerUnitAmount:          alpacadecimal.NewFromFloat(5),
}
line = line.AddDiscountForOverage(rating.AddDiscountInput{
	BilledAmountBeforeLine: prev,
	MaxSpend:               maxSpend,
// ...
```

<!-- archie:ai-end -->
