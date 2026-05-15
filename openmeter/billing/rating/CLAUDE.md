# rating

<!-- archie:ai-start -->

> Public contract layer for the billing rating sub-system: defines the Service interface, DetailedLine value type, accessor interfaces (StandardLineAccessor, GatheringLineAccessor), and ChildUniqueReferenceID constants consumed by the rating/service implementation, charges, and billing worker. Callers import only this package — never rating/service.

## Patterns

**ChildUniqueReferenceID constants from const.go** — All child line identifiers must use the named constants in const.go (e.g. UsageChildUniqueReferenceID, UnitPriceUsageChildUniqueReferenceID). Never hardcode these strings in pricers, mutators, or callers. (`ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID`)
**currencyx.Calculator for all monetary arithmetic** — Every amount on DetailedLine must be rounded via currencyx.Calculator.RoundToPrecision. Raw alpacadecimal arithmetic is only acceptable as an intermediate step before storage. (`total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))`)
**Functional options for GenerateDetailedLines variants** — Behavioural knobs (IgnoreMinimumCommitment, DisableCreditsMutator) are passed as variadic GenerateDetailedLinesOption functions. Never add boolean flags directly to the Service interface signature. (`svc.GenerateDetailedLines(line, rating.WithMinimumCommitmentIgnored())`)
**Accessor interfaces — never concrete billing line types** — Callers supply lines via StandardLineAccessor or GatheringLineAccessor. The rating package must not import or reference concrete billing.InvoiceLine structs directly. (`func (s *service) GenerateDetailedLines(in StandardLineAccessor, opts ...GenerateDetailedLinesOption) (GenerateDetailedLinesResult, error)`)
**AddDiscountForOverage for max-spend capping** — Maximum-spend cap discounts must be applied by calling DetailedLine.AddDiscountForOverage with an AddDiscountInput; returns a new value. Never compute overage discounts inline in pricers or service methods. (`line = line.AddDiscountForOverage(rating.AddDiscountInput{BilledAmountBeforeLine: prev, MaxSpend: max, Currency: calc})`)
**DetailedLine is an immutable value type** — All With-style mutations (AddDiscountForOverage) return a new DetailedLine copy. Pricers must produce new values rather than mutating shared state. No pointer receivers that mutate in place. (`updated := line.AddDiscountForOverage(input) // original line unchanged`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `const.go` | Single source of truth for all ChildUniqueReferenceID string values used by pricers, mutators, tests, and callers to label DetailedLine children. | Adding a new price type without a corresponding constant here causes silent mismatches in child-line correlation across charges and billing worker. |
| `detailedline.go` | Defines the DetailedLine value type, TotalAmount computation (PerUnitAmount×Quantity minus discounts and credits), AddDiscountForOverage, and DetailedLines.Sum. | TotalAmount already subtracts AmountDiscounts and CreditsApplied — double-subtracting in callers produces incorrect invoice totals. Negative Quantity is valid (usage correction); negative PerUnitAmount is rejected by Validate(). |
| `line.go` | Accessor interfaces (StandardLineAccessor, GatheringLineAccessor, PriceAccessor) that decouple the rating package from concrete billing line types. | Adding methods to these interfaces forces all implementors to update; prefer adding helpers in service.go or rating/service instead. |
| `service.go` | Service interface, GenerateDetailedLinesResult, ResolveBillablePeriodInput (with Validate()), and functional option types. Concrete implementation lives in rating/service sub-package. | ResolveBillablePeriodInput.Validate() is the authoritative check — do not duplicate this validation in callers or the implementation. |

## Anti-Patterns

- Hardcoding ChildUniqueReferenceID strings instead of using the constants in const.go
- Setting input.Usage for FlatPriceType lines — pricer validation returns an error
- Computing DetailedLine totals (rounding, discount subtraction) outside TotalAmount or getTotalsFromDetailedLines in rating/service
- Importing concrete billing line types (billing.InvoiceLine) directly in this package — use the accessor interfaces in line.go
- Adding state fields to the service struct in rating/service — the implementation is intentionally stateless; New() returns an empty struct

## Decisions

- **Service interface lives in the rating package root; implementation in rating/service sub-package** — Callers (charges, billing worker) import only the interface and types without pulling in implementation dependencies such as pricer logic and mutator chains.
- **DetailedLine is a value type with copy-on-write semantics for all mutations** — Immutable-by-convention pipeline: pricers produce new DetailedLine values rather than mutating shared state, making mutation order explicit and preventing aliasing bugs in the priceMutator chain.
- **Behavioural knobs use functional options (GenerateDetailedLinesOption) rather than input struct fields** — Keeps the primary input path (StandardLineAccessor) stable while allowing opt-in overrides (IgnoreMinimumCommitment, DisableCreditsMutator) without breaking existing callers.

## Example: Computing a DetailedLine total with max-spend overage discount

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/invopop/gobl/currency"
)

calc, _ := currencyx.Code(currency.USD).Calculator()
line := rating.DetailedLine{
	Name:                   "Unit usage",
	ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
	Quantity:               alpacadecimal.NewFromFloat(10),
	PerUnitAmount:          alpacadecimal.NewFromFloat(5),
}
line = line.AddDiscountForOverage(rating.AddDiscountInput{
	BilledAmountBeforeLine: prevTotal,
// ...
```

<!-- archie:ai-end -->
