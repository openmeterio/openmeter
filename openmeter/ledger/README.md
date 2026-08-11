# Ledger

The ledger is OpenMeter's immutable accounting journal for customer credit,
receivables, accrued usage, recognized earnings, payments, currency conversion,
and breakage.

## Domain model

Accounts express ownership and accounting purpose. Subaccounts are the actual
posting addresses; a subaccount's normalized route is part of its identity.

Customer accounts are provisioned per customer. Business accounts are shared
within a namespace.

| account | meaning |
| --- | --- |
| `customer_fbo` | customer credit or stored value; not a literal regulated FBO bank account |
| `customer_receivable` | value owed by the customer, including open and payment-authorized stages |
| `customer_accrued` | acknowledged usage or spend not yet recognized as earnings |
| `wash` | the external payment or cash boundary |
| `earnings` | recognized business revenue |
| `brokerage` | business-side counterparty for currency conversion |
| `breakage` | expired or otherwise forfeited customer credit |

The historical ledger stores immutable transaction groups, transactions, and
entries. Balances are projections over those entries; they are not mutable
facts stored independently from the journal.

## Ownership and boundaries

- The ledger owns balanced posting, route identity, permitted accounting flows,
  collection from concrete balances, corrections, and accounting provenance.
- Transaction templates encode posting mechanics. They do not decide charge
  lifecycle, payment lifecycle, settlement mode, or invoice orchestration.
- `chargeadapter` translates charge lifecycle events into ledger templates.
  Charges decide when an effect is due; the ledger decides how that effect is
  represented and validated. See the [charges domain](../billing/charges/README.md).
- The collector owns source selection and correction unwind order. Callers
  provide the target amount and attribution; they must not recreate collection
  policy.
- `customerbalance` is a customer-facing projection over booked ledger state,
  breakage, and not-yet-booked charge impacts. It does not own posting,
  collection, or correction rules.
- Account resolvers provision canonical customer and namespace business
  accounts. Higher-level domains request account-specific route parameters
  rather than assembling generic accounts or posting addresses.

## Transaction invariants

- Every transaction sums to zero. Each entry amount is valid at the posting
  currency's precision.
- `CommitGroup` validates the whole input, locks every affected parent account,
  and books the group atomically in the caller's database transaction.
- Default routing rules constrain allowed account-type combinations, flow
  direction, authorization stages, route compatibility, and dimension scope.
  A balanced transaction can still be invalid.
- Reversal and correction logic follows the actual original entries. It
  preserves charge provenance and route pairing, links replacement postings to
  their source entries, and uses deterministic source order rather than
  recomputing an idealized replacement from current balances.
- Entry identity records collection or correction linkage and source and spend
  charge provenance when present. Template codes and transaction annotations
  describe accounting meaning.
- Credit-backed earnings recognition consumes only accrued buckets whose
  source credit and spend charge are both present and distinct. Accrued value
  with no source charge, or with the same source and spend charge, is
  invoice-backed and remains deferred.

The historical ledger makes a group atomic, but it does not deduplicate a
repeated `CommitGroup` call. The initiating domain must make retries safe and
persist the returned group reference with its own lifecycle state. Ledger
annotations and entry identity preserve accounting meaning and provenance; they
are not operation idempotency keys.

## Route invariants

Routes currently carry currency, feature restrictions, cost basis, credit
priority, receivable authorization status, tax code, and tax behavior. These
are accounting identity, not optional metadata. Dropping a populated dimension
during translation or filtering can merge economically distinct balances while
leaving each transaction locally balanced.

- Routes are normalized before key creation and querying. Feature order is not
  semantic.
- `Route.Filter()` pins present values, including explicit nil values.
  `RouteFilter` absence means "do not filter"; it differs from filtering for a
  nil route dimension.
- Feature dimensions belong only on FBO and receivable routes. Tax dimensions
  belong only on accrued and earnings routes; FBO sources acquire the charge's
  tax attribution when value moves into accrued.
- Currency and cost-basis attribution survive the relevant FBO, receivable,
  accrued, earnings, wash, brokerage, and breakage legs.
- Credit priority controls FBO collection order. Corrections unwind the
  recorded collection order rather than applying today's priorities.
- Receivable authorization is an accounting stage. Moving between open and
  authorized routes is a ledger transaction, not an in-place flag update.

Adding or changing a route dimension is a storage and compatibility change. It
affects normalization, routing-key versions, schema persistence, filters,
account-specific route parameters, transaction rules, corrections, and
historical data—not only the `Route` struct.

## Feature-restricted balances

Public balance filtering and source allocability are related but distinct:

- no feature filter means the whole credit portfolio
- an unrestricted-only filter selects routes with no feature restriction
- filtering for one feature includes unrestricted routes and routes containing
  that feature
- unrestricted credit can fund any charge; restricted credit can fund only a
  matching feature

Public filtering cannot be implemented as exact route equality, and a public
balance result is not necessarily the set of sources allocable to every charge.

## Time and balance boundaries

- `BookedAt` is accounting effective time. `CreatedAt` and transaction ID make
  ordering deterministic when booked times tie.
- `AsOf` and cursor behavior apply before customer-facing projection; future
  ledger or breakage facts cannot leak into historical views.
- `historical.Ledger` enforces transaction invariance, not post-transaction
  account-balance constraints. Product-specific bounds, including when a
  balance may go negative, belong to higher-level flows and collectors.
