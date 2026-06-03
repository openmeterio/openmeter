# currencyx

<!-- archie:ai-start -->

> Thin ISO 4217 currency wrapper on invopop/gobl/currency adding a Calculator for currency-precision rounding and largest-remainder allocation helpers. Used across billing and subscription for all monetary arithmetic to ensure correct subunit rounding per currency.

## Patterns

**Validate before Calculator** — Always call Code.Validate() or obtain a Calculator via Code.Calculator() (which validates internally). Never use a Code value without validation. (`calc, err := currencyx.Code("USD").Calculator(); if err != nil { return err }`)
**Calculator.RoundToPrecision for monetary amounts** — Round alpacadecimal.Decimal to the correct subunit count for the currency. Never hardcode decimal places. (`rounded := calc.RoundToPrecision(amount) // USD -> 2 places, JPY -> 0 places`)
**Largest-remainder allocation at currency precision** — Use AllocateByWeight (WeightedAllocationInput) or AllocateByAmount (AmountAllocationInput) to split an amount across keys; both round at the currency precision and distribute the remainder by largest fractional remainder, with CompareKey as a deterministic tie-breaker. (`out, err := currencyx.AllocateByWeight(calc, currencyx.WeightedAllocationInput[string]{Amount: total, Items: items, CompareKey: cmp.Compare})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.go` | Code type (alias of currency.Code), Validate, Calculator constructor, RoundToPrecision using Def.Subunits, IsRoundedToPrecision. | RoundToPrecision uses Def.Subunits (not smallestDenomination) — correct for online payments; physical-cash rounding may differ. |
| `allocation.go` | Generic largest-remainder allocation: WeightedAllocationItem/AmountAllocationItem inputs and AllocateByWeight/AllocateByAmount returning per-key amounts that sum exactly to the input amount. | Zero input amount returns nil; CompareKey ties are broken deterministically — supply it (e.g. cmp.Compare) when item order is not stable. |

## Anti-Patterns

- Rounding monetary amounts with a fixed precision constant instead of Calculator.RoundToPrecision.
- Using currency.Code (gobl) directly instead of currencyx.Code — the wrapper adds Validate() and Calculator().
- Hand-splitting amounts proportionally instead of AllocateByWeight/AllocateByAmount — manual splits drop the rounding remainder.

## Decisions

- **Wraps gobl/currency rather than implementing ISO 4217 from scratch.** — GOBL is already a transitive dependency for invoicing; reusing its currency definitions avoids a duplicate definition table.

## Example: Round a billing amount to currency precision

```
import (
  "github.com/openmeterio/openmeter/pkg/currencyx"
  "github.com/alpacahq/alpacadecimal"
)

calc, err := currencyx.Code("USD").Calculator()
if err != nil { return err }
rounded := calc.RoundToPrecision(alpacadecimal.NewFromFloat(1.23456)) // -> 1.23
```

<!-- archie:ai-end -->
