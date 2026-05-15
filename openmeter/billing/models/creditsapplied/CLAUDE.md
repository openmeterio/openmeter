# creditsapplied

<!-- archie:ai-start -->

> Pure domain model package defining CreditsApplied (named slice type) and CreditApplied (value type) representing credits applied to billing line items. Enforces positive-amount invariant and provides currency-precision-aware summation. No persistence code lives here.

## Patterns

**models.Validator compile-time assertion** — Every type in this package must implement models.Validator via a Validate() error method and be verified with a compile-time assertion var _ models.Validator = (*T)(nil). (`var _ models.Validator = (*CreditsApplied)(nil)`)
**models.Clonable deep-copy via lo.Map** — CreditsApplied implements models.Clonable[CreditsApplied] using lo.Map to produce a safe shallow copy of the slice. Any new slice or pointer field must be deep-copied in Clone. (`func (c CreditsApplied) Clone() CreditsApplied { return lo.Map(c, func(item CreditApplied, _ int) CreditApplied { return item }) }`)
**Currency-aware summation with RoundToPrecision** — SumAmount rounds each item via currencyx.Calculator.RoundToPrecision before accumulating. New aggregation helpers must use the calculator, not raw Add. (`sum = sum.Add(currency.RoundToPrecision(item.Amount))`)
**Positive-amount enforcement in Validate** — CreditApplied.Validate() rejects non-positive amounts. Never bypass Validate before accumulating credits into billing totals. (`if !c.Amount.IsPositive() { return errors.New("amount must be positive") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Single source of all types in this package. Defines CreditsApplied, CreditApplied, and all methods including Clone, Validate, SumAmount, and CloneWithAmount. | CreditApplied.Amount must remain positive (validated). Adding nullable fields requires updating Clone and Validate. Do not add DB/Ent imports here. |

## Anti-Patterns

- Storing negative credit amounts — Validate() enforces positivity; bypassing it causes billing arithmetic errors.
- Accumulating amounts without RoundToPrecision — causes currency precision drift across line items.
- Adding DB/Ent concerns to this package — it is a pure domain model with no persistence code.
- Using == instead of .Equal() when comparing alpacadecimal.Decimal fields in any new helper.

## Decisions

- **CreditsApplied is a named slice type rather than a struct with a slice field.** — Allows direct iteration, len checks, and method attachment without wrapping, matching the pattern used by other billing model collections in this domain.

<!-- archie:ai-end -->
