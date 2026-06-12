# totals

<!-- archie:ai-start -->

> Money value-object aggregating the per-line/invoice monetary breakdown (Amount, ChargesTotal, DiscountsTotal, three tax totals, CreditsTotal, Total) plus an Ent mixin persisting each as a Postgres numeric. Provides decimal-safe arithmetic, validation, rounding, and DB Set/FromDB helpers reused across billing, charges, and rating.

## Patterns

**Immutable value-semantics decimal arithmetic** — Add/Sub/Neg/Sum/RoundToPrecision/CalculateTotal operate field-by-field with alpacadecimal methods and return new Totals (value receivers); never mutate in place. (`res.Amount = res.Amount.Add(other.Amount)`)
**Mixin Setter/Getter generic DB contract** — totals.Mixin declares 8 numeric fields; Set[T Setter[T]] writes them onto any Ent builder and FromDB reads them via TotalsGetter — the canonical reuse hook for line/invoice schemas. (`func Set[T Setter[T]](mut Setter[T], totals Totals) T { return mut.SetAmount(totals.Amount)... }`)
**Sequential non-negative Validate** — Validate returns the first negative-field error; ValidateTotalNonNegative is a cheaper Total-only check used where intermediate negatives are allowed. (`if t.Amount.IsNegative() { return errors.New("amount is negative") }`)
**CalculateTotal as the derived-total formula** — Total = Amount + ChargesTotal + TaxesExclusiveTotal - DiscountsTotal - CreditsTotal (credits and discounts subtracted); use this rather than re-deriving inline. (`alpacadecimal.Sum(t.Amount, t.ChargesTotal, t.TaxesExclusiveTotal, t.DiscountsTotal.Neg(), t.CreditsTotal.Neg())`)
**Currency-aware rounding** — RoundToPrecision rounds every field through a currencyx.Calculator before persistence/comparison. (`t.Amount = calc.RoundToPrecision(t.Amount)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Totals struct and all arithmetic/validation/rounding methods (Add, Sub, Neg, Sum, IsZero, Equal, RoundToPrecision, CalculateTotal, Validate, ValidateTotalNonNegative). | CreditsApplied/credits are pre-tax (CreditsTotal subtracted in CalculateTotal). Equal compares each decimal with .Equal — never ==. Any new field must be added to all 8-field methods consistently. |
| `mixin.go` | Ent mixin (8 numeric fields) + Setter/Getter interfaces + Set/FromDB generic helpers. | Field list here, the Totals struct, and Set/FromDB must stay in lockstep; all fields are dialect.Postgres numeric. |
| `model_test.go` | Round-trip rounding, Equal, and ValidateTotalNonNegative tests using USD calculator. | Asserts exact rounded strings (e.g. "10.01") — adding fields requires extending these assertions. |

## Anti-Patterns

- Adding a Totals field without updating every method (Add/Sub/Neg/IsZero/Equal/RoundToPrecision) plus mixin Fields/Setter/Getter — silent drift loses the field.
- Mutating a Totals receiver in place instead of returning a new value.
- Comparing Totals or its decimal fields with == / reflect.DeepEqual instead of Equal.
- Treating credits/discounts as additive in Total — CalculateTotal subtracts them.
- Persisting or comparing un-rounded decimals; round via currencyx.Calculator first.

## Decisions

- **Totals is a self-contained value-object with generic Setter/Getter DB hooks.** — Both billing lines/invoices and the charges/rating subsystems persist identical monetary breakdowns; one mixin + generics avoids duplicating 8 numeric columns and arithmetic per entity.
- **Credits are modeled pre-tax and subtracted in CalculateTotal.** — Matches the invoice math where credits reduce the taxable base alongside discounts.

<!-- archie:ai-end -->
