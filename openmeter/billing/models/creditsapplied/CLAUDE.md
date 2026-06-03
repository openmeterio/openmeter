# creditsapplied

<!-- archie:ai-start -->

> Pure domain model package defining CreditsApplied (named slice) and CreditApplied (value type) representing credits applied to billing line items — enforces a positive-amount invariant and currency-precision-aware summation. No persistence code lives here.

## Patterns

**models.Validator compile-time assertion** — Every type implements models.Validator via Validate() error and is verified with var _ models.Validator = (*T)(nil). (`var _ models.Validator = (*CreditsApplied)(nil)`)
**models.Clonable deep-copy via lo.Map** — CreditsApplied implements models.Clonable[CreditsApplied] via lo.Map; new slice/pointer fields must be deep-copied in Clone. (`func (c CreditsApplied) Clone() CreditsApplied { return lo.Map(c, func(item CreditApplied, _ int) CreditApplied { return item }) }`)
**Currency-aware summation with RoundToPrecision** — SumAmount rounds each item via currencyx.Calculator.RoundToPrecision before accumulating; new aggregation helpers must use the calculator. (`sum = sum.Add(currency.RoundToPrecision(item.Amount))`)
**Positive-amount enforcement in Validate** — CreditApplied.Validate() rejects non-positive amounts; never accumulate credits without validating. (`if !c.Amount.IsPositive() { return errors.New("amount must be positive") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Single source of CreditsApplied, CreditApplied and all methods (Clone, Validate, SumAmount, CloneWithAmount). | Amount must stay positive (validated). Adding nullable fields requires updating Clone and Validate. No DB/Ent imports here. |

## Anti-Patterns

- Storing negative credit amounts — Validate() enforces positivity.
- Accumulating amounts without RoundToPrecision — causes currency precision drift.
- Adding DB/Ent concerns — this is a pure domain model.
- Using == instead of .Equal() to compare alpacadecimal.Decimal fields.

## Decisions

- **CreditsApplied is a named slice type rather than a struct with a slice field.** — Allows direct iteration, len checks, and method attachment without wrapping, matching other billing model collections.

<!-- archie:ai-end -->
