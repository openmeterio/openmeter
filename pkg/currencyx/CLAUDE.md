# currencyx

<!-- archie:ai-start -->

> Thin ISO 4217 currency wrapper built on invopop/gobl/currency that adds a Calculator type for currency-precision decimal rounding, used across billing and subscription for monetary arithmetic.

## Patterns

**Validate before Calculator** — Always call Code.Validate() or obtain a Calculator via Code.Calculator() which validates internally. Never use a Code without validation. (`calc, err := currencyx.Code("USD").Calculator(); if err != nil { return err }`)
**Calculator.RoundToPrecision for monetary amounts** — Round alpacadecimal.Decimal to the correct subunit count for the currency using Calculator.RoundToPrecision. Never hardcode decimal places. (`rounded := calc.RoundToPrecision(amount)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.go` | Code type (type alias of currency.Code), Validate, Calculator constructor, RoundToPrecision, IsRoundedToPrecision. | RoundToPrecision uses Def.Subunits (not smallest denomination) — fine for online payments but not for physical cash. |

## Anti-Patterns

- Rounding monetary amounts with a fixed precision constant instead of Calculator.RoundToPrecision
- Using currency.Code (gobl) directly instead of currencyx.Code — the wrapper adds Validate and Calculator

## Decisions

- **Wraps gobl/currency rather than implementing ISO 4217 from scratch** — GOBL is already a transitive dependency for invoicing; reusing its currency definitions avoids a duplicate definition table.

<!-- archie:ai-end -->
