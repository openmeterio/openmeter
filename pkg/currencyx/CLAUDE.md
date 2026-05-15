# currencyx

<!-- archie:ai-start -->

> Thin ISO 4217 currency wrapper on invopop/gobl/currency that adds a Calculator type for currency-precision decimal rounding. Used across billing and subscription for all monetary arithmetic to ensure correct subunit rounding per currency.

## Patterns

**Validate before Calculator** — Always call Code.Validate() or obtain a Calculator via Code.Calculator() which validates internally. Never use a Code value without validation. (`calc, err := currencyx.Code("USD").Calculator(); if err != nil { return err }`)
**Calculator.RoundToPrecision for monetary amounts** — Round alpacadecimal.Decimal to the correct subunit count for the currency. Never hardcode decimal places. (`rounded := calc.RoundToPrecision(amount) // USD -> 2 places, JPY -> 0 places`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.go` | Code type (alias of currency.Code), Validate, Calculator constructor, RoundToPrecision using Def.Subunits, IsRoundedToPrecision. | RoundToPrecision uses Def.Subunits (not smallestDenomination) — correct for online payments, but physical-cash rounding may differ. |

## Anti-Patterns

- Rounding monetary amounts with a fixed precision constant instead of Calculator.RoundToPrecision
- Using currency.Code (gobl) directly instead of currencyx.Code — the wrapper adds Validate() and Calculator()

## Decisions

- **Wraps gobl/currency rather than implementing ISO 4217 from scratch** — GOBL is already a transitive dependency for invoicing; reusing its currency definitions avoids a duplicate definition table.

## Example: Round a billing amount to currency precision

```
import (
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/alpacahq/alpacadecimal"
)

calc, err := currencyx.Code("USD").Calculator()
if err != nil {
	return err
}
rounded := calc.RoundToPrecision(alpacadecimal.NewFromFloat(1.23456)) // -> 1.23
```

<!-- archie:ai-end -->
