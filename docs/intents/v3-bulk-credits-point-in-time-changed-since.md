# Intent: bulk operational credit reads (v3)

Status: in review · Owner: Metering & Billing product

## Problem

Integrations currently read credit grants and balances one customer at a time.
Namespace-wide routes reduce client requests while retaining the existing credit
read semantics.

## Scope

| Route | Behavior |
| --- | --- |
| `GET /openmeter/credits/grants` | Lists grants across customers with the existing grant filters and sort, plus `filter[customer_id]`. Pages are capped at 1000. |
| `GET /openmeter/credits/balances` | Lists current balances for non-deleted customers in customer-ID order. Supports customer, currency, and feature filters. Pages are capped at 100. |

Grants include a read-only `customer` reference. Balance reads run for up to
8 customers concurrently. Customers without ledger accounts have empty balances;
a partial account setup or another read failure fails the page.

These are operational reads. Customers are read independently, and concurrent
changes can affect balances and pagination. They do not provide an accounting-close
snapshot or incremental synchronization. The existing per-customer historical
balance API remains unchanged.

## Deferred

- Historical bulk balances and point-in-time charge rerating.
- `updated_at` filtering for incremental synchronization.
- Namespace-wide credit transactions and transaction date filters.
- Periodic partial realization, accounting reporting, and period close.

Accounting reporting requires persisted realization results and correction
handling. Period close additionally needs completion and posting-boundary rules,
including how later adjustments relate to a closed period. Bulk reads do not add
those lifecycle capabilities.
