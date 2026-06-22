---
name: custom-currencies
description: Custom currency coordination for OpenMeter. Use when changes touch custom currency codes or fiat/custom boundaries in `pkg/currencyx`, `openmeter/currencies`, ledger routes/transactions/balances, charge settlement, product catalog or subscription mapping, schema migrations, or credit tests.
---

# Custom Currencies

Use this skill to keep custom-currency work boundary-driven. Use only public repository sources in code, tests, comments, migrations, committed docs, and review notes; do not cite uncommitted planning notes, branch URLs, or non-public business context.

Also load the package skill for each touched area: `ledger`, `charges`, `billing`, `subscription`, `api`, `ent`, `db-migration`, and `test`.

## Boundary Model

- **Currency code**: can be fiat or namespace-scoped custom. Custom codes are durable business identifiers, so historical finance facts must stay readable after display metadata changes or archive.
- **Product boundary**: validates registry semantics before new effects are created, including active definition, archive rules, ISO collision rejection, and product-facing errors.
- **Finance boundary**: resolves fiat basis, snapshots/copies basis context, and performs fiat materialization rounding.
- **Ledger boundary**: preserves durable text currency, posted decimal amount, route dimensions, and balanced transaction invariants without live registry lookup.
- **Invoice boundary**: keeps invoice money-of-account fiat. Custom units may appear as metadata or description, not as invoice currency.
- **Charge boundary**: keeps each charge denominated in exactly one currency. Multi-currency subscriptions are mapping and validation work, not a new charge lifecycle.

## Package Surfaces

- `pkg/currencyx`: code shape, fiat validation/calculator, allocation helpers, schema constants.
- `openmeter/currencies`: custom currency definitions, cost basis, service, adapter.
- `openmeter/ledger`: route validation, routing keys, account dimensions, historical ledger, transactions, balances, breakage.
- `openmeter/ledger/chargeadapter`: ledger-backed credit purchase, flat fee, usage-based handlers.
- `openmeter/billing/charges`: charge lifecycle, settlement modes, realization runs, allocation/correction, line engines.
- `openmeter/billing`, `openmeter/billing/models/stddetailedline`: invoice and detailed-line currency storage, fiat materialization.
- `openmeter/productcatalog`, `openmeter/subscription`, `openmeter/billing/worker/subscriptionsync`: rate-card currency and subscription-to-charge mapping.
- `openmeter/ent/schema`, `tools/migrate/migrations`: schema source and migrations for currency fields and finance context.
- `test/credits`: cross-stack sanity tests for ledger-backed customer credit behavior.

## Process

1. **Name the surface before editing.** Pick one primary surface: registry, cost basis, ledger fact, funding, catalog/subscription mapping, charge settlement, balance visibility, or breakage. Continue only when the scope can be stated in one sentence.
2. **Classify every changed path by boundary.** For each package you will edit, write down whether it is product, finance, ledger, invoice, or charge/subscription boundary work. Continue only when validation and rounding ownership are clear.
3. **Keep unrelated surfaces out.** If a path merely carries `currencyx.Code`, do not expand scope unless the request requires that boundary. Continue only when each edited file is justified by the named surface.
4. **Preserve fiat-only behavior.** Keep existing fiat calculator, precision, invoice currency, tax, payment, and lifecycle behavior unless the request explicitly changes it. Continue only when fiat regression coverage is selected or the reason for omitting it is clear.
5. **Verify the slice.** Run focused unit tests for validation/math and integration tests for ledger-backed credit behavior when balances, transactions, or charge adapters change. Finish only after reporting commands run or why they could not run.

## Ledger Checks

- Accept structurally valid custom codes in `ledger.Route` and routing key generation.
- Reject invalid codes: empty, too short, too long, whitespace-padded, or containing the route delimiter.
- Preserve fiat precision checks for fiat codes.
- Preserve exact posted decimal amounts for custom codes unless an upstream materialization boundary normalized them.
- Keep transaction groups balanced per currency.
- Use linked single-currency legs for fiat-to-custom funding; never put two currencies into one entry.
- Verify balance queries filter by custom currency and discover custom currencies with activity.
- Add replay/idempotency coverage when the same funding event can run more than once.

## Charges And Billing Checks

- Keep charge lifecycle generic over currency where possible.
- Burn same-currency customer credits for covered custom usage.
- Convert uncovered custom usage to fiat only at invoice materialization, using captured basis context.
- Do not use fiat calculators for custom-unit rounding unless the code is materializing fiat.
- Keep invoice line totals, tax, payment, and external invoicing fiat-denominated.
- Preserve charge status transitions and meta status synchronization when adding custom branches.

## Schema Checks

- Reuse shared currency code constants instead of hard-coding column widths.
- Widen every durable field that can store custom codes; do not widen only the first failing table.
- Generate Ent code and migrations from schema sources.
- Do not hand-edit generated Ent code.
- State narrowing/data-loss assumptions honestly in down migrations that reduce currency column width.

## Testing

Use direct commands. For Postgres-backed tests, set `POSTGRES_HOST=127.0.0.1`.

```bash
go test -count=1 -tags=dynamic ./pkg/currencyx ./openmeter/currencies
go test -count=1 -tags=dynamic ./openmeter/ledger
env POSTGRES_HOST=127.0.0.1 go test -count=1 -tags=dynamic ./openmeter/ledger/...
env POSTGRES_HOST=127.0.0.1 go test -count=1 -tags=dynamic ./openmeter/ledger/chargeadapter ./openmeter/ledger/customerbalance
env POSTGRES_HOST=127.0.0.1 go test -count=1 -tags=dynamic ./test/credits
```

For schema work, also follow the `ent` and `db-migration` skills.

## Review Phrases

- "Custom currency codes are accepted as durable ledger route values."
- "Custom balances are visible through ledger-backed balance reads."
- "Paid fiat-to-custom funding is implemented" only when linked fiat/custom ledger legs, basis context, and idempotency are covered.
- "Custom charge settlement is implemented" only when covered custom usage and uncovered fiat materialization are both covered.
- "Storage supports long custom codes" only means persistence width is ready; it does not imply registry-backed precision or product archive rules.
