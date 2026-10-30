# Intent: bulk credit reads, point-in-time reads, and changed-since filters (v3)

Status: in review · Owner: Metering & Billing product

## Problem

Many integrations copy v3 billing data into a data warehouse. They load it on a
schedule and take a daily or month-end close snapshot of credit balances and
charge totals for revenue reporting.

Three gaps make this slow and fragile:

1. **No bulk credit routes.** Credits can only be read one customer at a time.
   A full credit snapshot needs several requests per customer. Its run time
   grows with the customer count, and a snapshot that takes too long misses its
   close window.
2. **No point-in-time read.** Live charge totals have no history. If the
   snapshot is missed, the close cannot be rebuilt later.
3. **No changed-since filter.** Every list route must be fully re-read on each
   sync and compared on the client side.

The goal is to make a missed close snapshot recoverable and to cut the requests
a full sync needs.

## What changes

### Namespace-wide credit routes

| Route | Behavior |
| --- | --- |
| `GET /openmeter/credits/grants` | Lists credit grants for all customers. Has the same filters and sort as the per-customer route, plus `filter[customer_id]`. |
| `GET /openmeter/credits/balances` | Lists credit balances for each customer, one page of customers at a time, ordered by customer ID. Supports `filter[customer_id]`, `filter[currency]`, `filter[feature_key]` and `timestamp`. |

- Credit grants now include a read-only `customer` reference on every route,
  so rows from the bulk list can be matched to a customer.
- The balances route works out each customer's balance on the server, several
  customers at a time. The page size is capped at 100. One request covers up to
  100 customers, instead of one or more requests per customer.
- A customer with no credit account gets an empty `balances` list instead of
  an error.
- **Deferred: `GET /openmeter/credits/transactions`.** Running balances in the
  ledger are kept per customer. Expiry and void records are loaded per
  customer. The ledger has no index from account to customer. A correct bulk
  route needs new ledger queries, so it is a separate change.

### Point-in-time reads

**Balances.** Both balance routes accept `timestamp`. This read already worked
on the per-customer route. It returns `booked`, `spendable` and `pending` as of
that time, based on all ledger postings recorded now. Re-running it later
returns the same `booked` value unless an entry is booked with an earlier
timestamp, such as a correction or a grant with a past `effective_at`, so
`booked` is the figure to close on. `spendable` at a past time can
still drop afterwards, because it sets aside credit for usage recorded later.

`live` stays `0` for past timestamps. Usage that was not yet booked at a past
time cannot be rebuilt.

**A close no longer has to run at the close time.** A close job can read
`booked` any time afterwards, for example with
`timestamp=<day>T23:59:59.999999Z`. The cutoff is inclusive and has microsecond
precision.

**Charge totals.** Both charge list routes accept `as_of`. It can be used only
together with `expand=real_time_usage`, and it must not be in the future.

The response contains the `usage` and `totals.realtime` values the live read
would have returned at `as_of`. Only events stored before `as_of` count, and
minimum commitments are applied only once the service period has ended at
`as_of`.

The meter part of the result is exact and can be repeated. Events that arrive
late are excluded, the same way they were excluded from a live read at that
time.

| Reflects current state, not state at `as_of` | Why |
| --- | --- |
| Price, discounts and service period | Charges are edited in place and have no history. |
| Charge status and realizations | Same reason. |

Charges created after `as_of` are still returned. Clients can drop them by
`created_at`.

### `updated_at` filter on list routes

Add `filter[updated_at]` (`eq`, `gt`, `gte`, `lt`, `lte`) to these routes:

| Route | Change detection covers |
| --- | --- |
| `GET /openmeter/billing/invoices` | Invoice header |
| `GET /openmeter/customers` | Customer |
| `GET /openmeter/subscriptions` | Subscription row |
| `GET /openmeter/plans` | Plan row |
| `GET /openmeter/charges`, `GET /openmeter/customers/{id}/charges` | Charge row |
| `GET /openmeter/customers/{id}/credits/grants`, `GET /openmeter/credits/grants` | Grant row |

`updated_at` changes when that resource's own record changes. These changes do
not update it:

- edits to child records: invoice lines, subscription phases and items, plan
  rate cards, charge realizations;
- live usage moving on an open charge.

The API docs say this. Updating the invoice's `updated_at` when its lines change
is a separate change.

Credit transactions are not included. Ledger rows are never changed after they
are written, so they have no `updated_at`. Changed-since reads for the ledger
need `booked_at` and `created_at` filters, which are a separate change.

## Out of scope

- Credit transaction date filters, invoice list ordering and `expand`
  improvements, gateway limits, and strict handling of unknown query
  parameters.
- Database indexes for namespace-wide grant ordering. Add an index if query
  plans show it is needed.

## Expected impact

| Workload | Before | After |
| --- | --- | --- |
| Credit close snapshot | Several requests per customer, so run time grows with customer count and must fit the close window | About two requests per 100 customers (grants and balances). Can run any time after the close. The ledger is still read per customer until bulk transactions ship. |
| Incremental collection sync | Full re-read of every list route | Only rows changed since the last sync (`updated_at`). Open invoices and charges still need a live re-read; see the caveats above. |

## Delivery

The work ships as three independent pull requests that can merge in any order:

1. **Namespace-wide credit routes** (this document lives here): the bulk grant
   and balance routes, point-in-time balances, the grant `customer` reference,
   the `updated_at` filter on credit grants, and the Go SDK naming fix that
   keeps the per-customer grant params type unchanged.
2. **`updated_at` filters**: invoices, customers, subscriptions, plans, and
   charges.
3. **Point-in-time charge totals**: `as_of` on both charge list routes.

## Open questions for review

- Should namespace-wide routes stay marked `x-unstable` like their
  per-customer versions? Yes, for now.
- Should `as_of` on charges also hide charges created after `as_of`? The
  proposal says no: the filter stays explicit and the client decides.
- Is the balances page cap of 100 right behind the Konnect gateway timeout?
  Each customer's balance is computed on request, 8 customers at a time. Past
  timestamps skip the live part and are faster.
