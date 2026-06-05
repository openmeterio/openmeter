# creditsapplied

<!-- archie:ai-start -->

> Tiny value-object package modeling credits applied to a billing line. Defines CreditApplied (Amount/Description/CreditRealizationID) and the CreditsApplied slice, persisted as JSONB on detailed-line schemas and pre-tax in the totals model.

## Patterns

**Validator + Clonable interface assertions** — Compile-time interface checks via var _ blank assignments enforce that the type satisfies models.Validator and models.Clonable. (`var _ models.Validator = (*CreditsApplied)(nil); var _ models.Clonable[CreditsApplied] = (*CreditsApplied)(nil)`)
**Slice-level Validate delegates to element Validate** — CreditsApplied.Validate() ranges items and returns the first item.Validate() error; CreditApplied.Validate() rejects non-positive Amount. (`if !c.Amount.IsPositive() { return errors.New("amount must be positive") }`)
**Value-semantics clone helpers** — Clone() returns nil for empty slices and lo.Map copies elements; CloneWithAmount returns a copy with a new Amount (value receiver, no mutation of original). (`func (c CreditApplied) CloneWithAmount(amount alpacadecimal.Decimal) CreditApplied { c.Amount = amount; return c }`)
**Currency-aware summation** — SumAmount rounds each item Amount with currencyx.Calculator.RoundToPrecision before adding, accumulating on alpacadecimal.Zero. (`sum = sum.Add(currency.RoundToPrecision(item.Amount))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Entire package: CreditApplied struct, CreditsApplied slice, Validate/Clone/SumAmount/CloneWithAmount. | Amount must be a positive alpacadecimal.Decimal — never compare decimals with ==; use alpaca methods. Clone returns nil (not empty slice) for len==0. |

## Anti-Patterns

- Adding monetary fields without rounding through a currencyx.Calculator before summing.
- Returning an empty non-nil slice from Clone() instead of nil — downstream goderive equality treats nil vs empty distinctly.
- Mutating the receiver in CloneWithAmount/Clone (value receivers exist to keep these copy-only).

## Decisions

- **Credits are stored as a standalone value-object slice rather than line fields.** — Lets the same CreditsApplied JSONB blob be reused across detailed-line schemas and summed independently for pre-tax totals.

<!-- archie:ai-end -->
