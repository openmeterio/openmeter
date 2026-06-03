# totals

<!-- archie:ai-start -->

> Defines the Totals struct — a billing line's financial breakdown (amount, taxes, charges, discounts, credits, total) with currency-precision rounding, additive aggregation (Add/Sum), subtraction, negation, and an Ent mixin persisting all 8 fields as Postgres numeric columns. Used by all billing line types and invoice aggregations.

## Patterns

**Ent mixin with typed Setter[T]/TotalsGetter** — totals.Mixin declares all 8 numeric fields; use Set[T](mut, totals) and FromDB(e) for Ent interactions, never individual Set*/Get* in adapters. (`func Set[T Setter[T]](mut Setter[T], totals Totals) T { return mut.SetAmount(totals.Amount).SetTaxesTotal(totals.TaxesTotal) /* all 8 */ }`)
**RoundToPrecision before persist/compare** — RoundToPrecision applies calculator.RoundToPrecision to every field atomically, returning a new Totals; always call before persisting or cross-currency comparison. (`func (t Totals) RoundToPrecision(calc currencyx.Calculator) Totals`)
**Additive aggregation via Add/Sum** — Totals.Add accumulates field-by-field; Sum(...) aggregates a variadic list. Never accumulate individual fields outside these helpers. (`func Sum(others ...Totals) Totals { return Totals{}.Add(others...) }`)
**Non-negative invariant in Validate** — All 8 fields must be non-negative; new fields need a non-negative check in Validate. (`if t.Amount.IsNegative() { return errors.New("amount is negative") }`)
**Fixed CalculateTotal formula** — CalculateTotal = Amount + ChargesTotal + TaxesExclusiveTotal - DiscountsTotal - CreditsTotal; never recompute inline. (`func (t Totals) CalculateTotal() alpacadecimal.Decimal { return alpacadecimal.Sum(t.Amount, t.ChargesTotal, t.TaxesExclusiveTotal, t.DiscountsTotal.Neg(), t.CreditsTotal.Neg()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Totals struct with Validate, Add, Sub, Neg, Sum, Equal, IsZero, RoundToPrecision, CalculateTotal. | A new field requires updating Add, Sub, Neg, RoundToPrecision, Equal, IsZero, Validate, and CalculateTotal (if in the total formula). |
| `mixin.go` | Ent mixin + Setter[T]/TotalsGetter interfaces + Set[T]/FromDB helpers. | All fields use dialect.Postgres 'numeric'. A new field updates Fields(), Setter[T], TotalsGetter, Set[T], FromDB in lockstep — all five. |
| `model_test.go` | Unit tests for RoundToPrecision (USD half-up) and Equal correctness. | Tests use exact string comparison on rounded decimals — update expected values if precision logic changes. |

## Anti-Patterns

- Calling individual Set*/Get* methods for totals fields in adapters — use Set[T]/FromDB.
- Accumulating totals outside Add/Sum.
- Skipping RoundToPrecision before persistence.
- Adding a field without updating all of Add/Sub/Neg/Equal/IsZero/RoundToPrecision/Validate/mixin Fields()/Setter[T]/TotalsGetter/Set[T]/FromDB.

## Decisions

- **All totals fields use alpacadecimal.Decimal stored as Postgres numeric.** — Billing requires exact decimal arithmetic; float64 SQL types would introduce storage rounding errors.
- **Mixin and domain model co-located in one package.** — Keeps Setter/Getter interfaces in sync with the struct fields — a field addition immediately surfaces missing interface methods.

<!-- archie:ai-end -->
