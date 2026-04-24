# totals

<!-- archie:ai-start -->

> Defines the Totals struct representing a billing line's financial breakdown (amount, taxes, charges, discounts, credits, total) with currency-precision rounding, additive aggregation, and an Ent mixin for persisting all fields as PostgreSQL numeric columns.

## Patterns

**Ent mixin with typed Setter/Getter interfaces** — totals.Mixin declares all 8 numeric DB fields. totals.Setter[T] and totals.TotalsGetter provide typed interfaces for Ent create/update mutations and DB reads. Use Set[T](mut, totals) and FromDB(e) for all Ent interactions — never call individual Set*/Get* methods directly. (`func Set[T Setter[T]](mut Setter[T], totals Totals) T { return mut.SetAmount(totals.Amount). /* chain all 8 fields */ }`)
**Currency-precision rounding via currencyx.Calculator** — RoundToPrecision applies calculator.RoundToPrecision to every field atomically. Always call this before persisting or comparing totals across currencies. (`func (t Totals) RoundToPrecision(calc currencyx.Calculator) Totals`)
**Additive aggregation via Add/Sum** — Totals.Add accumulates multiple Totals values field-by-field. Use Sum(...) to aggregate a variadic list. Do not accumulate individual fields outside these helpers. (`func Sum(others ...Totals) Totals { return Totals{}.Add(others...) }`)
**Validate enforces non-negative invariants** — All 8 fields must be non-negative. Validate() is called in the billing domain before persistence. New fields added to Totals must have a corresponding non-negative check in Validate. (`if t.Amount.IsNegative() { return errors.New("amount is negative") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Totals struct with Validate, Add, Sum, RoundToPrecision, and CalculateTotal methods. | CalculateTotal formula: Amount + ChargesTotal + TaxesExclusiveTotal - DiscountsTotal - CreditsTotal. Adding a new field requires updating Add, RoundToPrecision, Validate, and CalculateTotal as appropriate. |
| `mixin.go` | Ent mixin (Mixin struct) + Setter[T]/TotalsGetter interfaces + Set[T] and FromDB helpers. | All 8 fields use dialect.Postgres: 'numeric'. Adding a field requires updating Fields(), Setter[T], TotalsGetter, Set[T], and FromDB in lockstep. |
| `model_test.go` | Unit test for RoundToPrecision with USD half-up rounding semantics. | Tests use exact string comparison on rounded decimals — update expected values if precision logic changes. |

## Anti-Patterns

- Directly calling individual Set*/Get* Ent methods for totals fields — always use Set[T] and FromDB helpers for consistency.
- Accumulating totals outside Add/Sum — leads to fields being missed.
- Skipping RoundToPrecision before persistence — causes numeric precision drift in multi-currency invoices.
- Adding a field to Totals without updating Add, RoundToPrecision, Validate, mixin Fields(), Setter[T], TotalsGetter, Set[T], and FromDB.

## Decisions

- **All totals fields use alpacadecimal.Decimal stored as PostgreSQL numeric.** — Billing requires exact decimal arithmetic; float64 SQL types would introduce rounding errors at the storage layer.
- **Mixin and domain model co-located in the same package.** — Ensures the Setter/Getter interfaces stay in sync with the Totals struct fields without cross-package indirection.

<!-- archie:ai-end -->
