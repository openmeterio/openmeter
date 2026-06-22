---
name: custom-currencies
description: Custom currency and prepaid credit coordination for OpenMeter. Use when changes touch custom currency codes, cost basis, credit grants, settlement modes, negative balances, breakage, or fiat/custom boundaries in `pkg/currencyx`, `openmeter/currencies`, ledger routes/transactions/balances, charge settlement, product catalog or subscription mapping, schema migrations, or credit tests.
---

# Custom Currencies

Use this skill to keep custom-currency work boundary-driven. Use only public repository sources in code, tests, comments, migrations, committed docs, and review notes; do not cite uncommitted planning notes, branch URLs, or non-public business context.

Also load the package skill for each touched area: `ledger`, `charges`, `billing`, `subscription`, `api`, `ent`, `db-migration`, and `test`.

## Boundary Model

- **Currency code**: can be fiat or namespace-scoped custom. Custom codes are durable business identifiers, so historical finance facts must stay readable after display metadata changes or archive.
- **Product boundary**: validates registry semantics before new effects are created, including active definition, archive rules, ISO collision rejection, and product-facing errors.
- **Finance boundary**: resolves fiat basis, snapshots/copies basis context, and performs fiat materialization rounding.
- **Ledger boundary**: preserves durable text currency, posted decimal amount, route dimensions, and balanced transaction invariants without live registry lookup.
- **Invoice boundary**: keeps invoice money-of-account fiat. Custom/non-fiat units must not be stored as billing invoice currency; they may appear only as metadata or description after fiat materialization.
- **Charge boundary**: keeps each charge denominated in exactly one currency. Multi-currency subscriptions are mapping and validation work, not a new charge lifecycle.
- **Credit instrument boundary**: defines grants, spendable units, validity, restrictions, priority, and auditable balance views.
- **Settlement boundary**: decides whether uncovered usage becomes an invoice overage or negative credit exposure.
- **Accounting boundary**: records funding, deferred revenue, recognized revenue, receivables/exposure, tax decisions, reversals, and breakage without conflating them with payment-provider events.

## Package Surfaces

- `pkg/currencyx`: code shape, fiat validation/calculator, allocation helpers, schema constants.
- `openmeter/currencies`: custom currency definitions, cost basis, service, adapter.
- `openmeter/ledger`: route validation, routing keys, account dimensions, historical ledger, transactions, balances, breakage.
- `openmeter/ledger/chargeadapter`: ledger-backed credit purchase, flat fee, usage-based handlers.
- `openmeter/billing/charges`: charge lifecycle, settlement modes, realization runs, allocation/correction, line engines.
- `openmeter/billing`, `openmeter/billing/models/stddetailedline`: fiat-only invoice artifacts and fiat materialization. The shared detailed-line mixin may be used by charge run tables too; only billing invoice instantiations must stay fiat-only.
- `openmeter/productcatalog`, `openmeter/subscription`, `openmeter/billing/worker/subscriptionsync`: rate-card currency and subscription-to-charge mapping.
- `openmeter/ent/schema`, `tools/migrate/migrations`: schema source and migrations for currency fields and finance context.
- `test/credits`: cross-stack sanity tests for ledger-backed customer credit behavior.

## Process

1. **Name the surface before editing.** Pick one primary surface: registry, cost basis, credit grant, ledger fact, funding, settlement mode, negative balance resolution, catalog/subscription mapping, charge settlement, balance visibility, or breakage. Continue only when the scope can be stated in one sentence.
2. **Classify every changed path by boundary.** For each package you will edit, write down whether it is product, finance, ledger, invoice, or charge/subscription boundary work. Continue only when validation and rounding ownership are clear.
3. **Keep unrelated surfaces out.** If a path merely carries `currencyx.Code`, do not expand scope unless the request requires that boundary. Continue only when each edited file is justified by the named surface.
4. **Preserve fiat-only behavior.** Keep existing fiat calculator, precision, invoice currency, tax, payment, and lifecycle behavior unless the request explicitly changes it. Continue only when fiat regression coverage is selected or the reason for omitting it is clear.
5. **Verify the slice.** Run focused unit tests for validation/math and integration tests for ledger-backed credit behavior when balances, transactions, or charge adapters change. Finish only after reporting commands run or why they could not run.

## Prepaid Credit Model

- Keep credit instrument, settlement, and accounting separate. A grant defines spendable units; settlement resolves rated usage; accounting records funding, revenue, tax, receivables/exposure, and breakage.
- Support fiat credits and custom credits. A custom credit grant may be purchased or invoiced in fiat, so store credit currency/amount separately from purchase currency/amount.
- Treat `allocated` and `cleared` as auditable persisted grant balances. Treat `available`, `reserved`, `projected`, and `consumed` as derived views unless schema explicitly says otherwise.
- Only active grants contribute to available, reserved, and projected balances. Cancelled, reversed, expired, or depleted grants can remain visible for audit but must not be spendable.
- Burn grants by scoped match, lower numeric priority, earliest expiry, then earliest creation.

## Cost Basis Checks

- Model cost basis as an effective-dated auditable mapping from `(credit_currency, base_fiat_currency)` to a rate, scoped to namespace and optionally customer.
- Snapshot or copy basis context whenever custom credits become fiat amounts: uncovered usage invoices, negative-balance invoices, paid top-ups, accounting/reporting valuation, and corrections.
- Do not recompute historical fiat value from the live registry after invoice, funding, finalization, or correction. Keep immutable basis fields such as snapshot id, base currency, rate, and as-of time.
- Keep cost basis optional until a custom currency crosses a fiat materialization boundary. Fail at product or finance validation, not ledger balance reads.
- Keep custom-unit arithmetic as exact posted decimal amounts. Apply fiat rounding only when materializing fiat.

## Settlement Checks

- For `invoice` settlement, burn eligible same-currency credits first, then invoice uncovered usage as overage; do not leave invoice-mode overage as credit debt.
- For `credit_only` settlement, settle all usage to the credit ledger. If credits are insufficient, allow negative credit balance/exposure and defer invoicing until a resolution policy runs.
- Keep settlement mode independent from funding source, payment state, and accounting policy.
- Negative-balance policies can leave exposure open, invoice at period end using a cost basis snapshot, auto top-up by creating and funding a grant, or enforce a hard limit only when runtime enforcement exists.
- For late or amended usage across finalized invoices, add explicit correction/reversal entries instead of mutating invoice-linked ledger facts.

## Funding And Accounting Checks

- `credit_grant` creates the credit container and should not move deferred revenue unless the business flow is explicitly already funded.
- Track provider payment state as append-only updates. Authorization is a funds check and must not require an invoice id.
- Use a separate funding event as the canonical accounting point that moves deferred revenue and links the purchase invoice.
- Use distinct invoice fields: `purchase_invoice_id` for stored-value purchases and `period_invoice_id` for usage invoices. Do not copy purchase invoice ids onto unrelated usage entries.
- Usage should reduce deferred revenue for covered value and recognize revenue for delivered service. Uncovered `credit_only` usage creates receivable/exposure until resolved.
- Model expiration and breakage as explicit auditable ledger entries. Expiring unused credits should not create a usage invoice.

## Ledger Checks

- Accept structurally valid custom codes in `ledger.Route` and routing key generation.
- Reject invalid codes: empty, too short, too long, whitespace-padded, or containing the route delimiter.
- Preserve fiat precision checks for fiat codes.
- Preserve exact posted decimal amounts for custom codes unless an upstream materialization boundary normalized them.
- Keep transaction groups balanced per currency.
- Use linked single-currency legs for fiat-to-custom funding; never put two currencies into one entry.
- Verify balance queries filter by custom currency and discover custom currencies with activity.
- Add replay/idempotency coverage when the same funding event can run more than once.
- Keep grant eligibility and burn-down time rules explicit. Do not mix billing-period eligibility with per-event timestamp eligibility in the same flow.

## Charges And Billing Checks

- Keep charge lifecycle generic over currency where possible.
- Burn same-currency customer credits for covered custom usage.
- Convert uncovered custom usage to fiat before creating billing invoice artifacts, using captured basis context.
- Do not use fiat calculators for custom-unit rounding unless the code is materializing fiat.
- Keep invoice line totals, tax, payment, and external invoicing fiat-denominated.
- Keep invoice calculation order stable: quantity, subtotal, discounts, prepaid credits, fiat materialization/conversion, tax, then customer balance.
- Preserve charge status transitions and meta status synchronization when adding custom branches.

## Schema Checks

- Reuse shared currency code constants instead of hard-coding column widths.
- Do not widen billing invoice tables for custom currency support. `billing_invoices`, `billing_invoice_lines`, `billing_invoice_split_line_groups`, and `billing_standard_invoice_detailed_lines` keep fiat-only currency columns.
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
- "Billing supports custom currencies" is wrong unless it means custom units are materialized before billing and billing persists only fiat invoice currency.
- "Storage supports long custom codes" only means persistence width is ready; it does not imply registry-backed precision or product archive rules.
