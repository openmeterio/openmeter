# rating

<!-- archie:ai-start -->

> Public contract layer for the billing rating sub-system: defines the Service interface, the immutable DetailedLine value type, accessor interfaces (StandardLineAccessor, GatheringLineAccessor, PriceAccessor), and the ChildUniqueReferenceID constants consumed by rating/service, charges, and the billing worker. Callers import only this package — never rating/service.

## Patterns

**ChildUniqueReferenceID constants from const.go** — All child-line identifiers must use the named constants in const.go (e.g. UsageChildUniqueReferenceID, UnitPriceUsageChildUniqueReferenceID, GraduatedTieredPriceUsageChildUniqueReferenceID). Never hardcode these strings in pricers, mutators, or callers. (`line := rating.DetailedLine{ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID}`)
**currencyx.Calculator for all monetary arithmetic** — Every amount on DetailedLine must be rounded via currencyx.Calculator.RoundToPrecision; TotalAmount() already does this. Raw alpacadecimal arithmetic is acceptable only as an intermediate step before rounding. (`total := in.Currency.RoundToPrecision(in.PerUnitAmount.Mul(in.Quantity))`)
**Functional options for GenerateDetailedLines variants** — Behavioural knobs (IgnoreMinimumCommitment, DisableCreditsMutator) pass as variadic GenerateDetailedLinesOption funcs (WithMinimumCommitmentIgnored, WithCreditsMutatorDisabled). Never add boolean flags directly to the Service interface signature. (`svc.GenerateDetailedLines(line, rating.WithMinimumCommitmentIgnored())`)
**Accessor interfaces — never concrete billing line types** — Callers supply lines via StandardLineAccessor / GatheringLineAccessor / PriceAccessor (line.go). The rating package decouples from concrete billing.InvoiceLine structs through these interfaces. (`func (s *service) GenerateDetailedLines(in StandardLineAccessor, opts ...GenerateDetailedLinesOption) (GenerateDetailedLinesResult, error)`)
**DetailedLine is an immutable value type (copy-on-write)** — All mutations (AddDiscountForOverage) return a new DetailedLine copy; the original is unchanged. Pricers must produce new values rather than mutating shared state — no pointer receivers that mutate in place. (`updated := line.AddDiscountForOverage(input) // original line unchanged`)
**AddDiscountForOverage for max-spend capping** — Maximum-spend cap discounts are applied by calling DetailedLine.AddDiscountForOverage(AddDiscountInput{BilledAmountBeforeLine, MaxSpend, Currency}); it returns a new line with a MaximumSpend AmountLineDiscount. Never compute overage discounts inline in pricers or service methods. (`line = line.AddDiscountForOverage(rating.AddDiscountInput{BilledAmountBeforeLine: prev, MaxSpend: max, Currency: calc})`)
**Validate() at construction boundaries** — DetailedLine.Validate() rejects negative PerUnitAmount, empty ChildUniqueReferenceID, and empty Name (negative Quantity is allowed for usage corrections). ResolveBillablePeriodInput.Validate() is the authoritative period-input check — do not duplicate it in callers. (`if err := line.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `const.go` | Single source of truth for all ChildUniqueReferenceID string values used to label DetailedLine children across pricers, mutators, tests, and callers. | Adding a new price type without a corresponding constant here causes silent child-line correlation mismatches across charges and the billing worker. Some IDs are format strings (e.g. GraduatedTieredPriceUsageChildUniqueReferenceID) — fill in the %d. |
| `detailedline.go` | DetailedLine value type, TotalAmount (PerUnitAmount x Quantity minus AmountDiscounts and CreditsApplied), AddDiscountForOverage, DetailedLines.Sum. | TotalAmount already subtracts AmountDiscounts and CreditsApplied — double-subtracting in callers produces wrong totals. Negative Quantity is valid; negative PerUnitAmount is rejected by Validate(). |
| `line.go` | Accessor interfaces (PriceAccessor, StandardLineAccessor, GatheringLineAccessor) that decouple rating from concrete billing line types. | Adding methods to these interfaces forces every implementor to update; prefer helpers in service.go or rating/service instead. |
| `service.go` | Service interface (ResolveBillablePeriod, GenerateDetailedLines), GenerateDetailedLinesResult, Usage, ResolveBillablePeriodInput (with Validate()), and functional option types. Concrete impl lives in rating/service. | ResolveBillablePeriodInput.Validate() is authoritative — do not re-validate in callers or the implementation. |

## Anti-Patterns

- Hardcoding ChildUniqueReferenceID strings instead of using the constants in const.go
- Setting input.Usage for FlatPriceType lines — pricer validation returns an error
- Computing DetailedLine totals (rounding, discount subtraction) outside TotalAmount or getTotalsFromDetailedLines in rating/service
- Importing concrete billing line types (billing.InvoiceLine) directly in this package — use the accessor interfaces in line.go
- Adding state fields to the service struct in rating/service — the implementation is intentionally stateless; New() returns an empty struct

## Decisions

- **Service interface lives in the rating package root; the implementation lives in the rating/service sub-package** — Callers (charges, billing worker) import only the interface and value types without pulling in pricer logic and mutator-chain dependencies.
- **DetailedLine is a value type with copy-on-write semantics for all mutations** — An immutable-by-convention pipeline lets pricers produce new DetailedLine values rather than mutating shared state, making mutation order explicit and preventing aliasing bugs in the priceMutator chain.
- **Behavioural knobs use functional options rather than input-struct fields** — Keeps the primary input path (StandardLineAccessor) stable while allowing opt-in overrides (IgnoreMinimumCommitment, DisableCreditsMutator) without breaking existing callers.

## Example: Compute a DetailedLine total with a max-spend overage discount

```
import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
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
// ...
```

<!-- archie:ai-end -->
