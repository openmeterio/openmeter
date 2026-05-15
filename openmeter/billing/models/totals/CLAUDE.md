# totals

<!-- archie:ai-start -->

> Defines the Totals struct representing a billing line's financial breakdown (amount, taxes, charges, discounts, credits, total) with currency-precision rounding, additive aggregation (Add/Sum), subtraction (Sub), negation (Neg), and an Ent mixin for persisting all 8 fields as PostgreSQL numeric columns. Used by all billing line types and invoice aggregations.

## Patterns

**Ent mixin with typed Setter[T]/TotalsGetter interfaces** — totals.Mixin declares all 8 numeric DB fields. Setter[T] and TotalsGetter provide typed interfaces for Ent create/update and DB reads. Use Set[T](mut, totals) and FromDB(e) for all Ent interactions — never call individual Set*/Get* methods directly in adapters. (`func Set[T Setter[T]](mut Setter[T], totals Totals) T { return mut.SetAmount(totals.Amount).SetTaxesTotal(totals.TaxesTotal). /* ... all 8 fields */ }`)
**Currency-precision rounding via RoundToPrecision** — RoundToPrecision applies calculator.RoundToPrecision to every field atomically, returning a new Totals value. Always call before persisting or comparing totals across currencies. (`func (t Totals) RoundToPrecision(calc currencyx.Calculator) Totals`)
**Additive aggregation via Add/Sum** — Totals.Add accumulates multiple Totals values field-by-field. Use Sum(...) to aggregate a variadic list. Do not accumulate individual fields outside these helpers to avoid missing a field. (`func Sum(others ...Totals) Totals { return Totals{}.Add(others...) }`)
**Non-negative invariant enforcement in Validate** — All 8 fields must be non-negative. Validate() is called in the billing domain before persistence. New fields added to Totals must have a corresponding non-negative check in Validate. (`if t.Amount.IsNegative() { return errors.New("amount is negative") }`)
**CalculateTotal formula is fixed** — CalculateTotal returns Amount + ChargesTotal + TaxesExclusiveTotal - DiscountsTotal - CreditsTotal. Never recompute this inline; always use CalculateTotal when deriving the final line total. (`func (t Totals) CalculateTotal() alpacadecimal.Decimal { return alpacadecimal.Sum(t.Amount, t.ChargesTotal, t.TaxesExclusiveTotal, t.DiscountsTotal.Neg(), t.CreditsTotal.Neg()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Totals struct with Validate, Add, Sub, Neg, Sum, Equal, IsZero, RoundToPrecision, and CalculateTotal methods. | Adding a new field requires updating Add, Sub, Neg, RoundToPrecision, Equal, IsZero, Validate, and CalculateTotal (if it participates in the total formula). All 8 existing methods must cover the new field. |
| `mixin.go` | Ent schema mixin (Mixin struct) + Setter[T]/TotalsGetter interfaces + Set[T] and FromDB helpers for Ent CRUD. | All fields use dialect.Postgres: 'numeric'. Adding a field requires updating Fields(), Setter[T], TotalsGetter, Set[T], and FromDB in lockstep — all five locations. |
| `model_test.go` | Unit tests for RoundToPrecision with USD half-up rounding semantics and Equal correctness. | Tests use exact string comparison on rounded decimals — update expected values if precision logic changes. Add analogous test cases for new fields. |

## Anti-Patterns

- Calling individual Set*/Get* Ent methods for totals fields in adapters — always use Set[T] and FromDB helpers.
- Accumulating totals outside Add/Sum — individual field accumulation leads to fields being missed.
- Skipping RoundToPrecision before persistence — causes numeric precision drift in multi-currency invoices.
- Adding a field to Totals without updating Add, Sub, Neg, Equal, IsZero, RoundToPrecision, Validate, mixin Fields(), Setter[T], TotalsGetter, Set[T], and FromDB.

## Decisions

- **All totals fields use alpacadecimal.Decimal stored as PostgreSQL numeric.** — Billing requires exact decimal arithmetic; float64 SQL types would introduce rounding errors at the storage layer.
- **Mixin and domain model co-located in the same package.** — Ensures the Setter/Getter interfaces stay in sync with the Totals struct fields without cross-package indirection — a field addition immediately surfaces missing interface methods.

<!-- archie:ai-end -->
