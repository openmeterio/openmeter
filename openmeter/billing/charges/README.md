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

### Customer charge override updates

Setting a customer charge override replaces the complete mutable override
snapshot while preserving the immutable base intent. Repeated sets update the
same layer; overrides never stack. Deletion state is not part of this operation
and remains owned by customer charge deletion.

Clearing a customer charge override removes that manual layer and restores the
latest reconciled base intent as effective. The operation is distinct from
charge deletion: it does not create or change an intent deletion marker. If the
restored base is live, lifecycle reconciliation can pass through the transient
`active.clear_override` detailed state, which owns any required reset before
dynamically selecting the restored `created`, `active`, or `final` lifecycle
state. Invoice-backed flat fees use this reset state for every live-base clear.
If the base was already deleted while hidden by a live override, clearing
instead passes through the transient `deleted.clear_override` state. That state
owns the same settlement-specific cleanup as system deletion and then
immediately advances to the usual `deleted` state. Both transient states
complete after their invoice effects are applied and lifecycle advancement
resumes. The base deletion timestamp is preserved, and the system
reconciliation deletion policy applies; clearing does not introduce a new
customer payment adjustment choice.

For invoice-backed flat fees, clearing an override that restores a live base
cancels the current realization line, including when it belongs to an immutable
issued or settled invoice, then detaches that run into audit history and starts
fresh billing work for the restored base. Existing prior runs remain untouched.
Billing may create a credit note for the deleted line; when it cannot, it records
unsupported correction history rather than blocking restoration.

Flat-fee override updates reuse normal rerating and invoice reconciliation.
Credit-only usage-based updates void and rebuild mutable realization history.
Invoice-backed usage-based updates are supported before realization starts;
after that point they are rejected until historical usage rerating and invoice
correction semantics are defined.

### Customer charge deletion payment adjustment

The customer charge API facade requires a payment adjustment when deleting a
flat-fee or usage-based charge. The facade owns this customer-facing choice and
maps it to the more detailed internal deletion policies used by charge state
machines.

`none` requests no compensating adjustment for economic effects already
realized by the charge. It maps to ignoring both credit and invoice refunds:
credit-only realizations are not corrected, and payment authorization,
settlement, payment intents, and external collection state are not refunded or
otherwise adjusted.

Payment adjustment is separate from invoice lifecycle reconciliation. Deleting
a charge still removes its gathering or mutable invoice lines. Immutable
invoice and payment history is preserved; unsupported drift is recorded as an
invoice validation issue.

## Lifecycle and persistence invariants

- Each concrete state machine is the authority for reachable detailed states.
  Every reachable status validates, and every detailed status maps to its short
  meta status.
- Invoice-backed charges remain in an `active.*` state while authorization or
  settlement is pending. `final` means the required payment lifecycle is
  complete, not merely that rating or line creation finished.
- Lifecycle entry points run in database transactions. Flat-fee and usage-based
  advancement and patching also take charge-scoped locks.
- Charge creation resolves every supplied feature reference before persistence.
  Usage-based features require a meter; flat-fee features are optional and may
  be meterless.
- Charge state machines do not expose invoice patches through a separate peek
  or drain operation. A caller must choose an invoice-aware
  `*UntilInvoicePatchesOrStable` operation, which stops at the first invoice
  effect boundary and transfers ownership of the returned patches, or a
  no-invoice-patches `*UntilStable` operation.
- The no-invoice-patches operations fail if a transition produces invoice
  patches. This makes forgetting to retrieve and apply invoice effects an
  explicit error instead of silently losing them while advancing lifecycle
  state.
- `UpdateCharge` persists the concrete base row. Expanded realizations are read
  model state and are written through dedicated adapter operations.
- Persisted charge discounts have stable correlation IDs allocated at their
  write boundary. Rerating, invoice projection, and correction reuse those IDs
  so discount child references preserve lineage across realization runs.
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
facts in order. An invoice-settled credit purchase has one standard invoice
line; duplicate lines for the same charge are rejected before lifecycle events
because the transition and its payment realization are bound to that line.

Payment-backed credit purchases also carry a charge-level cost basis. Fiat
credit uses a fixed scalar intent in the charge currency and materializes its
deterministic resolved state at charge creation. Custom-currency credit reuses
the shared manual, pinned, or dynamic cost-basis intent and its durable resolved
state; the shared model remains custom-currency-only. Dynamic intent is
persisted unresolved, then pinned to the rate effective at the purchase's
service-period start by the credit-purchase state machine. Externally settled
purchases resolve immediately before their credit grant. Invoice-settled
purchases enter billing with a provisional zero-value line; when billing creates
the standard invoice, the state machine resolves the cost basis before booking
the credit grant. The line engine then requires that resolution and replaces
the provisional amount with the resolved fiat value.

Persisted credit purchases read settlement and cost basis only from their
dedicated fields. Fiat purchases persist their scalar rate on the charge row;
custom-currency purchases reference durable shared cost-basis state. Resolution
time and charge creation time are not a validation invariant. The legacy
settlement JSON column is deprecated and ignored.

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
- Settlement-fiat overage allocation for a custom-currency usage-based or
  flat-fee run is a one-shot invoice-finalization effect. A persisted
  completion marker distinguishes pending allocation from a successful
  zero-allocation result, so retries do not re-enter completed allocation.
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
- allocates eligible settlement-fiat credits against the converted overage
- represents the converted overage as the invoice-line amount, applies
  settlement-fiat credits to that line, and leaves only the uncovered total
  collectible
- rounds the converted amount with the fiat currency
- retains the managed currency as charge identity rather than replacing it
  with the settlement fiat currency or display code

Charge-currency and settlement-fiat allocations are separate realization and
lineage domains. Rating and mutable rerating reconcile charge-currency facts;
invoice finalization first persists the gross converted overage, then allocates
settlement-fiat credits once against it. Invoice-issued callbacks only advance
the prepared custom-currency run. A line-finalization failure remains
retry-only. If synchronization with the invoicing app fails after preparation,
invoice deletion corrects settlement-fiat allocations, the gross overage, and
charge-currency allocations before deleting the prepared run. Cleanup and its
ledger effects share the billing transaction. Failed cleanup rolls back and
leaves preparation attached for retry from `delete.failed`; committed cleanup
sets `DeletedAt`, so later invoice-delete sync retries do not dispatch it again.

The converted post-allocation overage and the required fiat transaction are
separate settlement facts. A zero converted overage controls line omission and
invoice bypass. A zero required transaction controls payment handling and does
not by itself remove an otherwise billable line from the invoice.

This is a charge-domain contract, not end-to-end support: current ledger-backed
charge adapters reject custom currencies. The usage-based and flat-fee
settlement-fiat allocation and correction handlers remain explicit unsupported
stubs. Enabling them spans ledger route identity and persistence, charge
realization, settlement rounding, corrections, balance queries, and historical
migration.
