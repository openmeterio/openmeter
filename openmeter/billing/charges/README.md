# Charges

A charge is a durable billing intent plus the lifecycle state and realizations
created while fulfilling it. Charges connect subscription-derived or manually
created commercial intent to credit allocation, invoicing, payment, and ledger
effects.

## Domain model

The three charge types have distinct product behavior:

| type | intent |
| --- | --- |
| flat fee | realize a fixed, possibly prorated amount |
| usage based | meter and rate a service period, possibly through multiple realization runs |
| credit purchase | grant customer credit and track promotional, external, or invoice settlement |

The root `charges.Charge` is a tagged wrapper over concrete charge aggregates.
Each aggregate separates:

- intent: what should happen
- base state: current durable lifecycle and scheduling state
- realizations: persisted child facts such as runs, credit allocations,
  corrections, payments, detailed lines, and ledger transaction references

The shared meta status (`created`, `active`, `final`, `deleted`) is a query
projection. The type-specific detailed status is the lifecycle state.

## Ownership and boundaries

- The root charges service owns cross-type creation, lookup, listing, patch
  dispatch, and customer-scoped advancement.
- Flat-fee, usage-based, and credit-purchase packages own their lifecycle,
  detailed status, persistence, and settlement-specific behavior.
- State machines decide transitions and externally visible lifecycle effects.
  Realization subservices perform rating, allocation, correction, payment, or
  persistence mechanics without inventing transitions.
- Billing owns invoice aggregates, invoice creation, collection, and invoice
  lifecycle. Charge line engines translate charge state into billing lines and
  feed billing events back into the owning charge state machine.
- [Ledger charge adapters](../../ledger/README.md) translate requested economic
  effects into ledger transactions. They do not decide when a charge advances.
- [Subscription sync](../worker/subscriptionsync/README.md) reconciles
  subscription-derived source intent. It does not treat API overrides as new
  subscription source state.

`AdvanceCharges` coordinates concrete services; it is not a second
implementation of their state machines.

## Intent layers

Flat-fee and usage-based charges have an immutable intent, a mutable base layer,
and an optional mutable override layer.

- The base layer is the source intent. For subscription-managed charges it
  reflects subscription state.
- When present, the override is the effective customer-facing mutable intent.
- Customer-facing lifecycle, rating, invoice, and API behavior reads the
  effective layer.
- Subscription sync, repair, and adapter base persistence read the base layer.
- Immutable attribution such as customer, currency, subscription identity,
  feature identity, settlement mode, cost-basis intent, and tax configuration
  cannot silently move into an override.
- Override presence is represented by its dedicated row. Deprecated embedded
  override columns are not active behavior.
- Credit-purchase charges do not use this override model.

Patch source determines the target layer. System reconciliation targets source
state; API edits target an existing override, a manually managed base, or a new
override as appropriate. A state machine rejects a hidden base target while an
override is active. Service-level sync may update that hidden source state, but
does not emit invoice patches, rerate, or mutate customer-facing realizations
for it.

Effective deletion and base-intent deletion are therefore different query
concepts. Subscription reconciliation must be able to find a base intent even
when an active override hides it.

## Lifecycle and persistence invariants

- Each concrete state machine is the authority for reachable detailed states.
  Every reachable status validates, and every detailed status maps to its short
  meta status.
- Invoice-backed charges remain in an `active.*` state while authorization or
  settlement is pending. `final` means the required payment lifecycle is
  complete, not merely that rating or line creation finished.
- Lifecycle entry points run in database transactions. Flat-fee and usage-based
  advancement and patching also take charge-scoped locks.
- `UpdateCharge` persists the concrete base row. Expanded realizations are read
  model state and are written through dedicated adapter operations.
- Ledger effects and the realization or payment facts that reference them are
  written under the lifecycle transaction. Retry safety comes from persisted
  lifecycle facts checked before handlers run, not from the
  [ledger](../../ledger/README.md#transaction-invariants).
- Due `credit_only` flat-fee and usage-based charges are persisted before
  post-create auto-advance, so worker retries do not lose the intent. Credit
  purchases follow their own creation and invoice-event lifecycle.

## Settlement semantics

Flat-fee and usage-based charges support two modes:

- `credit_only` realizes against customer credit; live-balance projection treats
  its impact as unbounded and may therefore go negative
- `credit_then_invoice` consumes eligible positive credit and sends the
  remaining overage through billing

Their balance bounds, realizations, invoice behavior, and terminal conditions
are different; they are not interchangeable forms of one amount flow.

Credit purchases use a separate settlement union:

- promotional grants credit without payment
- external grants credit and tracks external authorization and settlement
- invoice settlement is driven by the charge-owned line engine, which feeds
  invoice-created, payment-authorized, and payment-settled billing events into
  the credit-purchase state machine

Payment-backed credit purchases grant credits before entering payment pending,
then record authorization and settlement as separate state-machine transitions.
Invoice settlement requires billing's authorization callback before settlement;
external settlement additionally supports a direct-paid path that records both
facts in order.

A credit grant, payment authorization, and payment settlement are separate
durable facts. A later state cannot be inferred from the presence of an earlier
one.

## Realization and time semantics

- Shrink and extend patches describe the direction of service coverage through
  `ServicePeriod.To`. Full-service, billing-period, and invoice timing reconcile
  to their target values independently; charge-type lifecycles use the service
  direction when deciding how to preserve or replace realizations.
- A usage-based realization run is a persisted checkpoint. `ServicePeriodTo`
  is the exclusive event-time bound; `StoredAtLT` is the exclusive ingestion
  bound. Waiting and correction logic uses the persisted bounds rather than
  recomputing them.
- Usage-based run metered quantity is cumulative from charge start to the run
  boundary. A billing standard line expects line-period and pre-line-period
  quantities, so charge mappers translate rather than copy it.
- Corrections reconcile against persisted allocations in the same realization
  run and monetary domain, preserving lineage to the facts previously billed
  or posted.
- Amount discounts on persisted detailed lines are signed realization facts.
  Their rounded amounts and rounding adjustments reconcile to the line's
  `DiscountsTotal`; correction lines can therefore carry negative discount
  values that reverse an earlier realization. Historical lines may not have a
  detailed discount breakdown; `DiscountsTotal` remains authoritative.
- Charge and billing detailed lines intentionally share only the standard line
  base. Charge discounts are immutable realization snapshots, while billing
  discounts are managed resources with their own identity and lifecycle, so
  the complete detailed-line model cannot be shared.
- Charge lifecycle timestamps are normalized to streaming aggregation
  precision before duration-sensitive calculations and persistence. Deletion
  timestamps retain their supplied instant.
- Charge-owned monetary inputs and totals are normalized with the charge
  currency. A positive source amount can round to zero after fiat conversion;
  that result requires explicit lifecycle handling.

The usage-based rating algorithms have narrower contracts in
[`delta`](usagebased/service/rating/delta/README.md),
[`periodpreserving`](usagebased/service/rating/periodpreserving/README.md), and
[`subtract`](usagebased/service/rating/subtract/README.md).

## Currency identity

`currencies.Currency` identity is the fiat code for fiat currencies and the
namespace plus managed currency ID for custom currencies. A display code is not
globally sufficient identity: distinct managed currencies may reuse one.

Within the charges domain, custom-currency `credit_then_invoice`:

- allocates and realizes in the custom currency first
- resolves and persists the applicable cost basis
- converts only the post-allocation overage to fiat
- rounds the converted amount with the fiat currency
- retains the managed currency as charge identity rather than replacing it
  with the settlement fiat currency or display code

This is a charge-domain contract, not end-to-end support: current ledger-backed
charge adapters reject custom currencies. Enabling them spans ledger route
identity and persistence, charge realization, settlement rounding, corrections,
balance queries, and historical migration.
