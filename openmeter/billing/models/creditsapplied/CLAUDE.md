# creditsapplied

<!-- archie:ai-start -->

> Defines the CreditsApplied slice type and CreditApplied value type representing credits applied to a billing line item. Enforces that all credit amounts are positive and provides currency-precision-aware summation.

## Patterns

**models.Validator interface compliance** — Both CreditsApplied and CreditApplied implement models.Validator via a Validate() error method. All new types here must do the same and be verified with a compile-time assertion: var _ models.Validator = (*T)(nil). (`var _ models.Validator = (*CreditsApplied)(nil)`)
**models.Clonable interface compliance** — CreditsApplied implements models.Clonable[CreditsApplied] with a Clone() method using lo.Map for safe copy. Any slice or pointer field added to this package must be deep-copied in Clone. (`func (c CreditsApplied) Clone() CreditsApplied { return lo.Map(c, func(item CreditApplied, _ int) CreditApplied { return item }) }`)
**Currency-aware summation** — SumAmount rounds each item via currencyx.Calculator.RoundToPrecision before accumulating. New aggregation helpers must use the calculator for rounding, not raw Add. (`sum = sum.Add(currency.RoundToPrecision(item.Amount))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Single source of all types in this package. Defines CreditsApplied, CreditApplied, and all methods. | CreditApplied.Amount must remain positive (validated). Adding nullable fields requires updating Clone and Validate. |

## Anti-Patterns

- Storing negative credit amounts — Validate() enforces positivity; bypassing it causes billing arithmetic errors.
- Accumulating amounts without RoundToPrecision — causes currency precision drift across line items.
- Adding DB/Ent concerns to this package — it is a pure domain model with no persistence code.

## Decisions

- **CreditsApplied is a named slice type rather than a struct with a slice field.** — Allows direct iteration, len checks, and method attachment without wrapping, matching the pattern used by other billing model collections.

<!-- archie:ai-end -->
